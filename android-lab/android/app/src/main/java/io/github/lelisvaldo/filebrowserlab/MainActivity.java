package io.github.lelisvaldo.filebrowserlab;

import android.Manifest;
import android.app.NotificationManager;
import android.content.Context;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.content.SharedPreferences;
import android.database.Cursor;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.provider.OpenableColumns;
import android.security.keystore.KeyGenParameterSpec;
import android.security.keystore.KeyPermanentlyInvalidatedException;
import android.security.keystore.KeyProperties;
import android.security.keystore.UserNotAuthenticatedException;
import android.util.Base64;
import android.view.View;
import android.view.Window;
import android.webkit.JavascriptInterface;
import android.webkit.ValueCallback;
import android.webkit.WebChromeClient;
import android.webkit.WebView;

import androidx.core.app.ActivityCompat;
import androidx.core.app.NotificationCompat;
import androidx.biometric.BiometricManager;
import androidx.biometric.BiometricPrompt;
import androidx.core.content.ContextCompat;
import androidx.core.view.WindowCompat;
import androidx.core.view.WindowInsetsControllerCompat;

import com.getcapacitor.BridgeActivity;
import com.getcapacitor.BridgeWebChromeClient;
import com.getcapacitor.BridgeWebViewClient;

import org.json.JSONArray;
import org.json.JSONObject;

import java.nio.charset.StandardCharsets;
import java.security.KeyFactory;
import java.security.KeyPairGenerator;
import java.security.KeyStore;
import java.security.PrivateKey;
import java.security.PublicKey;
import java.security.spec.MGF1ParameterSpec;
import java.security.spec.X509EncodedKeySpec;
import java.util.ArrayList;
import java.util.List;
import javax.crypto.Cipher;
import javax.crypto.spec.OAEPParameterSpec;
import javax.crypto.spec.PSource;

/**
 * Casca Android do filebrowser-lab: abre o servidor no WebView e acrescenta
 *  - bloqueio por biometria / PIN ao abrir (e depois de 30 s em segundo plano);
 *  - "Compartilhar para o servidor": os arquivos recebidos são entregues ao seletor de arquivos da página,
 *    que envia pelo mesmo caminho de sempre (painel de transferências, partes, progresso).
 */
public class MainActivity extends BridgeActivity {
    private static final long RELOCK_MS = 30_000;
    /** Arquivo recebido por Compartilhar ou lido de uma pasta escolhida; rel guarda o caminho dentro da pasta (null = só o nome). */
    private static final class Item {
        final String id;
        final Uri uri;       // arquivo recebido por Compartilhar (content://)
        final java.io.File file; // arquivo escolhido na tela de seleção do app
        final String rel;
        Item(String id, Uri uri, java.io.File file, String rel) { this.id = id; this.uri = uri; this.file = file; this.rel = rel; }
    }

    private final java.util.concurrent.CopyOnWriteArrayList<Item> shared = new java.util.concurrent.CopyOnWriteArrayList<>();
    private final java.util.concurrent.ConcurrentHashMap<String, Item> byId = new java.util.concurrent.ConcurrentHashMap<>();
    private final java.util.concurrent.atomic.AtomicInteger seq = new java.util.concurrent.atomic.AtomicInteger();
    private androidx.activity.result.ActivityResultLauncher<Intent> itemPicker;
    private final java.util.concurrent.CopyOnWriteArrayList<String> sharedDirs = new java.util.concurrent.CopyOnWriteArrayList<>(); // todas as pastas escolhidas (inclusive vazias)
    private long pausedAt = 0;
    private boolean locked = false;
    private boolean uploading = false;

    @Override
    public void onCreate(Bundle savedInstanceState) {
        final Thread.UncaughtExceptionHandler old = Thread.getDefaultUncaughtExceptionHandler();
        Thread.setDefaultUncaughtExceptionHandler((t, e) -> {
            try {
                java.io.StringWriter w = new java.io.StringWriter();
                e.printStackTrace(new java.io.PrintWriter(w));
                String txt = w.toString();
                getSharedPreferences("lab", MODE_PRIVATE).edit().putString("crash", txt.substring(0, Math.min(1500, txt.length()))).commit();
            } catch (Throwable ignored) { }
            if (old != null) old.uncaughtException(t, e);
        });
        super.onCreate(savedInstanceState);
        final String crash = prefs().getString("crash", null);
        if (crash != null) {
            prefs().edit().remove("crash").apply();
            new androidx.appcompat.app.AlertDialog.Builder(this)
                    .setTitle("O app fechou da última vez")
                    .setMessage(crash)
                    .setPositiveButton("OK", null)
                    .show();
        }
        WebView wv = getBridge().getWebView();
        getBridge().setWebViewClient(new BridgeWebViewClient(getBridge()) {
            @Override
            public android.webkit.WebResourceResponse shouldInterceptRequest(WebView view, android.webkit.WebResourceRequest req) {
                String p = req.getUrl().getPath();
                if (p != null && p.startsWith("/__labfile/")) return serveFile(p.substring(11), req.getRequestHeaders());
                return super.shouldInterceptRequest(view, req);
            }

            @Override
            public void onPageFinished(WebView view, String url) {
                super.onPageFinished(view, url);
                notifyShare();
            }
        });
        wv.setDownloadListener((url, ua, cd, mime, len) -> Downloader.start(MainActivity.this, url, ua, cd, mime));
        itemPicker = registerForActivityResult(new androidx.activity.result.contract.ActivityResultContracts.StartActivityForResult(), res -> {
            if (res.getResultCode() != RESULT_OK || res.getData() == null) return;
            final ArrayList<String> paths = res.getData().getStringArrayListExtra(PickerActivity.EXTRA_PATHS);
            if (paths == null || paths.isEmpty()) return;
            new Thread(() -> {
                List<Item> found = new ArrayList<>();
                List<String> dirs = new ArrayList<>();
                for (String p : paths) {
                    java.io.File f = new java.io.File(p);
                    if (f.isDirectory()) walk(f, f.getName(), found, dirs);
                    else if (f.isFile()) found.add(newItem(null, f, null));
                }
                runOnUiThread(() -> {
                    shared.addAll(found);
                    sharedDirs.addAll(dirs);
                    notifyShare();
                    // quem escolheu pela tela do app quer enviar já: o site começa o envio para a pasta aberta
                    getBridge().getWebView().evaluateJavascript("window.dispatchEvent(new Event('labpicked'))", null);
                });
            }, "lab-walk").start();
        });
        applyBars(prefs().getBoolean("dark", true)); // última cor conhecida do site, para não piscar ao abrir
        collectShare(getIntent());
        lock();
    }

    @Override
    protected void load() {
        // a ponte precisa existir antes da primeira página carregar (o Android só a injeta em páginas carregadas depois)
        WebView early = findViewById(com.getcapacitor.android.R.id.webview);
        if (early != null) early.addJavascriptInterface(new LabNative(), "LabNative");
        super.load();
    }

    @Override
    protected void onNewIntent(Intent intent) {
        super.onNewIntent(intent);
        setIntent(intent);
        collectShare(intent);
        notifyShare();
    }

    @Override
    public void onPause() {
        super.onPause();
        pausedAt = System.currentTimeMillis();
    }

    @Override
    public void onResume() {
        super.onResume();
        if (pausedAt > 0 && System.currentTimeMillis() - pausedAt > RELOCK_MS) lock();
        else if (locked) prompt();
    }

    // ---- cores: as barras do sistema seguem o tema do site (escuro / claro) ----
    private void applyBars(boolean dark) {
        Window w = getWindow();
        int bar = dark ? 0xFF120A33 : 0xFFDBDFE4;
        w.setStatusBarColor(bar);
        w.setNavigationBarColor(bar);
        WindowInsetsControllerCompat c = WindowCompat.getInsetsController(w, w.getDecorView());
        c.setAppearanceLightStatusBars(!dark);
        c.setAppearanceLightNavigationBars(!dark);
        getBridge().getWebView().setBackgroundColor(dark ? 0xFF170A38 : 0xFFE6E9ED);
    }

    // ---- bloqueio ----
    private SharedPreferences prefs() {
        return getSharedPreferences("lab", MODE_PRIVATE);
    }

    private int authenticators() {
        return Build.VERSION.SDK_INT >= 30
                ? BiometricManager.Authenticators.BIOMETRIC_STRONG | BiometricManager.Authenticators.DEVICE_CREDENTIAL
                : BiometricManager.Authenticators.BIOMETRIC_WEAK;
    }

    private boolean authenticating = false;

    // ---- entrar com biometria: usuário e senha guardados no cofre do Android (Keystore) ----
    // Chave RSA no Keystore: a pública cifra sem pedir biometria (na hora de ativar); a privada só decifra depois de uma
    // autenticação biométrica/PIN feita nos últimos 30 s (a mesma do bloqueio do app, então no uso normal é um toque só).
    private static final String KEY_ALIAS = "lab_login_v1";
    private boolean savedLoginPending = false;

    private boolean bioAvailable() {
        return BiometricManager.from(this).canAuthenticate(authenticators()) == BiometricManager.BIOMETRIC_SUCCESS;
    }

    private boolean hasSavedLogin() {
        return prefs().getString("savedLogin", null) != null;
    }

    private void clearSavedLoginData() {
        prefs().edit().remove("savedLogin").remove("savedUser").apply();
        try { KeyStore ks = KeyStore.getInstance("AndroidKeyStore"); ks.load(null); ks.deleteEntry(KEY_ALIAS); } catch (Exception ignored) { }
    }

    private OAEPParameterSpec oaep() {
        return new OAEPParameterSpec("SHA-256", "MGF1", MGF1ParameterSpec.SHA1, PSource.PSpecified.DEFAULT);
    }

    private boolean saveLoginData(String user, String pass) {
        try {
            byte[] plain = (user + "\n" + pass).getBytes(StandardCharsets.UTF_8);
            if (plain.length > 180) return false;
            clearSavedLoginData();
            KeyPairGenerator g = KeyPairGenerator.getInstance(KeyProperties.KEY_ALGORITHM_RSA, "AndroidKeyStore");
            KeyGenParameterSpec.Builder b = new KeyGenParameterSpec.Builder(KEY_ALIAS, KeyProperties.PURPOSE_ENCRYPT | KeyProperties.PURPOSE_DECRYPT)
                    .setKeySize(2048)
                    .setDigests(KeyProperties.DIGEST_SHA256, KeyProperties.DIGEST_SHA1)
                    .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_RSA_OAEP)
                    .setUserAuthenticationRequired(true);
            if (Build.VERSION.SDK_INT >= 30) b.setUserAuthenticationParameters(30, KeyProperties.AUTH_BIOMETRIC_STRONG | KeyProperties.AUTH_DEVICE_CREDENTIAL);
            else b.setUserAuthenticationValidityDurationSeconds(30);
            g.initialize(b.build());
            PublicKey pub = g.generateKeyPair().getPublic();
            PublicKey open = KeyFactory.getInstance(pub.getAlgorithm()).generatePublic(new X509EncodedKeySpec(pub.getEncoded()));
            Cipher c = Cipher.getInstance("RSA/ECB/OAEPWithSHA-256AndMGF1Padding");
            c.init(Cipher.ENCRYPT_MODE, open, oaep());
            prefs().edit().putString("savedLogin", Base64.encodeToString(c.doFinal(plain), Base64.NO_WRAP)).putString("savedUser", user).apply();
            return true;
        } catch (Exception e) {
            clearSavedLoginData();
            return false;
        }
    }

    /** Pede o login guardado: se o app ainda está travado, espera destravar; devolve à página pelo evento "labsavedlogin". */
    private void requestSavedLoginOnUi() {
        if (!hasSavedLogin()) { sendSavedLogin(null, null, "none"); return; }
        if (locked || authenticating) { savedLoginPending = true; return; }
        runSavedLogin();
    }

    private void runSavedLogin() {
        try {
            String r = decryptSaved();
            String[] p = r.split("\n", 2);
            sendSavedLogin(p[0], p.length > 1 ? p[1] : "", null);
        } catch (UserNotAuthenticatedException e) {
            promptForSavedLogin();
        } catch (KeyPermanentlyInvalidatedException e) {
            clearSavedLoginData();
            sendSavedLogin(null, null, "invalid");
        } catch (Exception e) {
            clearSavedLoginData();
            sendSavedLogin(null, null, "invalid");
        }
    }

    private String decryptSaved() throws Exception {
        KeyStore ks = KeyStore.getInstance("AndroidKeyStore");
        ks.load(null);
        PrivateKey key = (PrivateKey) ks.getKey(KEY_ALIAS, null);
        if (key == null) throw new KeyPermanentlyInvalidatedException();
        Cipher c = Cipher.getInstance("RSA/ECB/OAEPWithSHA-256AndMGF1Padding");
        c.init(Cipher.DECRYPT_MODE, key, oaep());
        return new String(c.doFinal(Base64.decode(prefs().getString("savedLogin", ""), Base64.NO_WRAP)), StandardCharsets.UTF_8);
    }

    private void promptForSavedLogin() {
        if (authenticating) { savedLoginPending = true; return; }
        authenticating = true;
        BiometricPrompt bp = new BiometricPrompt(this, ContextCompat.getMainExecutor(this), new BiometricPrompt.AuthenticationCallback() {
            @Override
            public void onAuthenticationSucceeded(BiometricPrompt.AuthenticationResult result) {
                authenticating = false;
                try {
                    String[] p = decryptSaved().split("\n", 2);
                    sendSavedLogin(p[0], p.length > 1 ? p[1] : "", null);
                } catch (Exception e) {
                    clearSavedLoginData();
                    sendSavedLogin(null, null, "invalid");
                }
            }

            @Override
            public void onAuthenticationError(int code, CharSequence msg) {
                authenticating = false;
                sendSavedLogin(null, null, "cancel");
            }
        });
        BiometricPrompt.PromptInfo.Builder info = new BiometricPrompt.PromptInfo.Builder()
                .setTitle("Filebrowser Lab")
                .setSubtitle("Entrar com a sua biometria")
                .setAllowedAuthenticators(authenticators());
        if (Build.VERSION.SDK_INT < 30) info.setNegativeButtonText("Cancelar");
        bp.authenticate(info.build());
    }

    private void sendSavedLogin(String user, String pass, String error) {
        final WebView wv = getBridge().getWebView();
        try {
            JSONObject o = new JSONObject();
            if (error != null) o.put("error", error); else o.put("user", user).put("pass", pass);
            final String js = "window.dispatchEvent(new CustomEvent('labsavedlogin',{detail:" + o.toString() + "}))";
            wv.post(() -> wv.evaluateJavascript(js, null));
        } catch (Exception ignored) { }
    }

    private void lock() {
        if (locked) return;
        if (!prefs().getBoolean("lock", true)) { afterUnlock(); return; }
        if (BiometricManager.from(this).canAuthenticate(authenticators()) != BiometricManager.BIOMETRIC_SUCCESS) { afterUnlock(); return; } // sem biometria/PIN configurado: não trava
        locked = true;
        getBridge().getWebView().setVisibility(View.INVISIBLE);
        prompt();
    }

    private void prompt() {
        if (authenticating || !locked) return;
        authenticating = true;
        final WebView wv = getBridge().getWebView();
        BiometricPrompt prompt = new BiometricPrompt(this, ContextCompat.getMainExecutor(this), new BiometricPrompt.AuthenticationCallback() {
            @Override
            public void onAuthenticationSucceeded(BiometricPrompt.AuthenticationResult result) {
                authenticating = false;
                locked = false;
                wv.setVisibility(View.VISIBLE);
                afterUnlock();
            }

            @Override
            public void onAuthenticationError(int code, CharSequence msg) {
                authenticating = false;
                // o usuário desistiu (ou errou demais): fecha o app. Qualquer outro cancelamento (por exemplo outro diálogo do
                // sistema por cima) mantém a tela travada e pede de novo quando o app voltar ao primeiro plano.
                if (code == BiometricPrompt.ERROR_USER_CANCELED || code == BiometricPrompt.ERROR_NEGATIVE_BUTTON
                        || code == BiometricPrompt.ERROR_LOCKOUT || code == BiometricPrompt.ERROR_LOCKOUT_PERMANENT) {
                    finishAndRemoveTask();
                }
            }
        });
        BiometricPrompt.PromptInfo.Builder info = new BiometricPrompt.PromptInfo.Builder()
                .setTitle("Filebrowser Lab")
                .setSubtitle("Confirme que é você para abrir")
                .setAllowedAuthenticators(authenticators());
        if (Build.VERSION.SDK_INT < 30) info.setNegativeButtonText("Cancelar");
        prompt.authenticate(info.build());
    }

    /** Depois de destravar: pede a permissão de notificações (uma vez) para o progresso dos envios e downloads. */
    private void afterUnlock() {
        if (savedLoginPending) { savedLoginPending = false; runSavedLogin(); }
        if (Build.VERSION.SDK_INT >= 33 && !prefs().getBoolean("askedNotif", false)
                && checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) {
            prefs().edit().putBoolean("askedNotif", true).apply();
            ActivityCompat.requestPermissions(this, new String[]{Manifest.permission.POST_NOTIFICATIONS}, 7);
        }
    }

    private Item newItem(Uri uri, java.io.File file, String rel) {
        Item it = new Item("f" + seq.incrementAndGet(), uri, file, rel);
        byId.put(it.id, it);
        return it;
    }

    /** Lê a pasta inteira (subpastas incluídas), guardando o caminho relativo de cada arquivo e todas as pastas, vazias ou não. */
    private void walk(java.io.File dir, String prefix, List<Item> out, List<String> dirs) {
        dirs.add(prefix);
        java.io.File[] kids = dir.listFiles();
        if (kids == null) return;
        for (java.io.File f : kids) {
            if (f.isDirectory()) walk(f, prefix + "/" + f.getName(), out, dirs);
            else if (f.isFile()) out.add(newItem(null, f, prefix + "/" + f.getName()));
        }
    }

    private java.io.InputStream openItem(Item it) throws java.io.IOException {
        return it.file != null ? new java.io.FileInputStream(it.file) : getContentResolver().openInputStream(it.uri);
    }

    private long sizeOf(Item it) {
        if (it.file != null) return it.file.length();
        try (Cursor c = getContentResolver().query(it.uri, null, null, null, null)) {
            if (c != null && c.moveToFirst()) {
                int s = c.getColumnIndex(OpenableColumns.SIZE);
                if (s >= 0 && !c.isNull(s)) return c.getLong(s);
            }
        } catch (Exception ignored) { }
        return -1;
    }

    /** Entrega os bytes de um arquivo escolhido ao site (/__labfile/<id>), aceitando Range para o envio em partes. */
    private android.webkit.WebResourceResponse serveFile(String id, java.util.Map<String, String> reqHeaders) {
        final Item it = byId.get(id);
        java.util.Map<String, String> h = new java.util.HashMap<>();
        h.put("Accept-Ranges", "bytes");
        if (it == null) return new android.webkit.WebResourceResponse("text/plain", "utf-8", 404, "Not Found", h, new java.io.ByteArrayInputStream(new byte[0]));
        try {
            long size = sizeOf(it);
            long start = 0, end = size - 1;
            boolean ranged = false;
            String rg = reqHeaders == null ? null : reqHeaders.get("Range");
            if (rg == null && reqHeaders != null) rg = reqHeaders.get("range");
            if (rg != null && rg.startsWith("bytes=") && size >= 0) {
                String[] pr = rg.substring(6).split("-", -1);
                if (!pr[0].isEmpty()) start = Long.parseLong(pr[0]);
                if (pr.length > 1 && !pr[1].isEmpty()) end = Math.min(size - 1, Long.parseLong(pr[1]));
                ranged = true;
            }
            java.io.InputStream in = openItem(it);
            long skipped = 0;
            while (skipped < start) {
                long n = in.skip(start - skipped);
                if (n <= 0) break;
                skipped += n;
            }
            final long len = size >= 0 ? Math.max(0, end - start + 1) : -1;
            if (len >= 0) h.put("Content-Length", String.valueOf(len));
            if (ranged) h.put("Content-Range", "bytes " + start + "-" + end + "/" + size);
            final java.io.InputStream src = in;
            java.io.InputStream limited = len < 0 ? src : new java.io.FilterInputStream(src) {
                long left = len;
                @Override public int read() throws java.io.IOException {
                    if (left <= 0) return -1;
                    int b = super.read();
                    if (b >= 0) left--;
                    return b;
                }
                @Override public int read(byte[] b, int off, int n) throws java.io.IOException {
                    if (left <= 0) return -1;
                    int r = super.read(b, off, (int) Math.min(n, left));
                    if (r > 0) left -= r;
                    return r;
                }
            };
            return new android.webkit.WebResourceResponse("application/octet-stream", null, ranged ? 206 : 200, ranged ? "Partial Content" : "OK", h, limited);
        } catch (Exception e) {
            return new android.webkit.WebResourceResponse("text/plain", "utf-8", 500, "Error", h, new java.io.ByteArrayInputStream(new byte[0]));
        }
    }

    // ---- compartilhar ----
    @SuppressWarnings({"deprecation", "unchecked"})
    private void collectShare(Intent i) {
        if (i == null) return;
        String a = i.getAction();
        if (Intent.ACTION_SEND.equals(a)) {
            Uri u = i.getParcelableExtra(Intent.EXTRA_STREAM);
            if (u != null) shared.add(newItem(u, null, null));
        } else if (Intent.ACTION_SEND_MULTIPLE.equals(a)) {
            ArrayList<Uri> l = i.getParcelableArrayListExtra(Intent.EXTRA_STREAM);
            if (l != null) for (Uri u : l) shared.add(newItem(u, null, null));
        }
    }

    private void notifyShare() {
        final WebView wv = getBridge().getWebView();
        wv.post(() -> wv.evaluateJavascript("window.dispatchEvent(new Event('labshare'))", null));
    }

    /** Ponte para a página do servidor (window.LabNative). */
    public class LabNative {
        @JavascriptInterface
        public String shareInfo() {
            JSONArray out = new JSONArray();
            JSONArray ds = new JSONArray();
            for (String d : sharedDirs) ds.put(d);
            try {
                for (Item it : shared) {
                    String name = it.file != null ? it.file.getName() : it.uri.getLastPathSegment();
                    if (it.file == null) {
                        try (Cursor c = getContentResolver().query(it.uri, null, null, null, null)) {
                            if (c != null && c.moveToFirst()) {
                                int n = c.getColumnIndex(OpenableColumns.DISPLAY_NAME);
                                if (n >= 0) name = c.getString(n);
                            }
                        } catch (Exception ignored) { }
                    }
                    out.put(new JSONObject().put("id", it.id).put("name", name).put("size", sizeOf(it)).put("rel", it.rel == null ? JSONObject.NULL : it.rel));
                }
            } catch (Exception ignored) { }
            try { return new JSONObject().put("files", out).put("dirs", ds).toString(); } catch (Exception e) { return "{}"; }
        }

        /** Abre a tela de seleção do app (pastas e arquivos na mesma lista); o que for marcado vira a lista de envio. */
        @JavascriptInterface
        public void pickItems() {
            runOnUiThread(() -> itemPicker.launch(new Intent(MainActivity.this, PickerActivity.class)));
        }

        @JavascriptInterface
        public void clearShare() {
            shared.clear();
            sharedDirs.clear();
            notifyShare();
        }

        @JavascriptInterface
        public boolean getLock() {
            return prefs().getBoolean("lock", true);
        }

        @JavascriptInterface
        public void setLock(boolean on) {
            prefs().edit().putBoolean("lock", on).apply();
        }

        @JavascriptInterface
        public void setBars(boolean dark) {
            prefs().edit().putBoolean("dark", dark).apply();
            runOnUiThread(() -> applyBars(dark));
        }

        /** Envios em andamento no site: mantém o app vivo em segundo plano e mostra o progresso numa notificação. */
        @JavascriptInterface
        public void uploadState(int count, int pct, String name, boolean paused) {
            Context c = getApplicationContext();
            if (!uploading) {
                uploading = true;
                TransferService.begin(c);
            }
            TransferService.ensureChannel(c);
            String title = count > 1 ? "Enviando " + count + " arquivos" : "Enviando " + name;
            NotificationCompat.Builder b = new NotificationCompat.Builder(c, TransferService.CHANNEL)
                    .setSmallIcon(android.R.drawable.stat_sys_upload).setContentTitle(title)
                    .setContentText(paused ? "Pausado" : pct + "%").setOngoing(true).setOnlyAlertOnce(true)
                    .setProgress(100, Math.max(0, Math.min(100, pct)), false).setContentIntent(TransferService.openApp(c));
            try { c.getSystemService(NotificationManager.class).notify(TransferService.FG_ID, b.build()); } catch (SecurityException ignored) { }
        }

        @JavascriptInterface
        public void uploadDone(int done, int failed) {
            Context c = getApplicationContext();
            if (uploading) {
                uploading = false;
                TransferService.end(c);
            }
            TransferService.ensureChannel(c);
            String text = failed > 0 ? failed + " não enviado(s), " + done + " enviado(s)" : done + (done == 1 ? " arquivo enviado" : " arquivos enviados");
            NotificationCompat.Builder b = new NotificationCompat.Builder(c, TransferService.CHANNEL)
                    .setSmallIcon(failed > 0 ? android.R.drawable.stat_notify_error : android.R.drawable.stat_sys_upload_done)
                    .setContentTitle("Envio concluído").setContentText(text).setAutoCancel(true).setContentIntent(TransferService.openApp(c));
            try { c.getSystemService(NotificationManager.class).notify(2, b.build()); } catch (SecurityException ignored) { }
        }

        /** Confere se o endereço é mesmo um servidor filebrowser-lab (a rota /api/lab/host responde JSON, 401 sem login). Responde por window.__labProbe(json). */
        @JavascriptInterface
        public void probe(final String base) {
            new Thread(() -> {
                String out;
                java.net.HttpURLConnection con = null;
                try {
                    con = (java.net.HttpURLConnection) new java.net.URL(base.replaceAll("/+$", "") + "/api/lab/host").openConnection();
                    con.setInstanceFollowRedirects(false);
                    con.setConnectTimeout(6000);
                    con.setReadTimeout(6000);
                    int code = con.getResponseCode();
                    String type = con.getContentType() == null ? "" : con.getContentType().toLowerCase();
                    boolean ok = type.contains("json") && (code == 401 || code == 403 || code == 200);
                    out = new JSONObject().put("ok", ok).put("code", code).toString();
                } catch (Exception e) {
                    try { out = new JSONObject().put("ok", false).put("code", 0).put("err", String.valueOf(e.getMessage())).toString(); } catch (Exception x) { out = "{\"ok\":false}"; }
                } finally {
                    if (con != null) con.disconnect();
                }
                final String res = out;
                final WebView w = getBridge().getWebView();
                w.post(() -> w.evaluateJavascript("window.__labProbe&&window.__labProbe(" + res + ")", null));
            }, "lab-probe").start();
        }

        /** Há biometria/PIN configurado no aparelho (dá para guardar o login)? */
        @JavascriptInterface
        public boolean bioAvailable() { return MainActivity.this.bioAvailable(); }

        @JavascriptInterface
        public boolean hasSavedLogin() { return MainActivity.this.hasSavedLogin(); }

        @JavascriptInterface
        public String savedUser() { return prefs().getString("savedUser", ""); }

        /** Guarda usuário e senha no cofre do Android, protegidos pela biometria. Devolve false se não deu. */
        @JavascriptInterface
        public boolean saveLogin(String user, String pass) { return saveLoginData(user == null ? "" : user, pass == null ? "" : pass); }

        @JavascriptInterface
        public void clearSavedLogin() { clearSavedLoginData(); }

        /** Pede o login guardado; a resposta chega pelo evento "labsavedlogin" (detail: {user, pass} ou {error}). */
        @JavascriptInterface
        public void requestSavedLogin() { runOnUiThread(MainActivity.this::requestSavedLoginOnUi); }

        @JavascriptInterface
        public void changeServer() {
            runOnUiThread(() -> getBridge().getWebView().loadUrl(getBridge().getLocalUrl() + "/index.html?trocar=1"));
        }
    }
}
