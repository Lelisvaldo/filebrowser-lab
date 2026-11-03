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

import java.util.ArrayList;
import java.util.List;

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
        final Uri uri;
        final String rel;
        Item(Uri uri, String rel) { this.uri = uri; this.rel = rel; }
    }

    private final List<Item> shared = new ArrayList<>();
    private androidx.activity.result.ActivityResultLauncher<Uri> folderPicker;
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
        wv.setWebChromeClient(new BridgeWebChromeClient(getBridge()) {
            @Override
            public boolean onShowFileChooser(WebView view, ValueCallback<Uri[]> cb, FileChooserParams params) {
                if (!shared.isEmpty()) {
                    Uri[] arr = new Uri[shared.size()];
                    for (int k = 0; k < arr.length; k++) arr[k] = shared.get(k).uri;
                    shared.clear();
                    cb.onReceiveValue(arr);
                    notifyShare();
                    return true;
                }
                return super.onShowFileChooser(view, cb, params);
            }
        });
        getBridge().setWebViewClient(new BridgeWebViewClient(getBridge()) {
            @Override
            public void onPageFinished(WebView view, String url) {
                super.onPageFinished(view, url);
                notifyShare();
            }
        });
        wv.setDownloadListener((url, ua, cd, mime, len) -> Downloader.start(MainActivity.this, url, ua, cd, mime));
        folderPicker = registerForActivityResult(new androidx.activity.result.contract.ActivityResultContracts.OpenDocumentTree(), tree -> {
            if (tree == null) return;
            new Thread(() -> {
                List<Item> found = new ArrayList<>();
                androidx.documentfile.provider.DocumentFile root = androidx.documentfile.provider.DocumentFile.fromTreeUri(MainActivity.this, tree);
                if (root != null) walk(root, root.getName() == null ? "pasta" : root.getName(), found);
                runOnUiThread(() -> {
                    shared.clear();
                    shared.addAll(found);
                    notifyShare();
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
        if (Build.VERSION.SDK_INT >= 33 && !prefs().getBoolean("askedNotif", false)
                && checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) {
            prefs().edit().putBoolean("askedNotif", true).apply();
            ActivityCompat.requestPermissions(this, new String[]{Manifest.permission.POST_NOTIFICATIONS}, 7);
        }
    }

    /** Lê a pasta escolhida inteira (subpastas incluídas), guardando o caminho relativo de cada arquivo. */
    private void walk(androidx.documentfile.provider.DocumentFile dir, String prefix, List<Item> out) {
        androidx.documentfile.provider.DocumentFile[] kids = dir.listFiles();
        for (androidx.documentfile.provider.DocumentFile f : kids) {
            String name = f.getName() == null ? "arquivo" : f.getName();
            if (f.isDirectory()) walk(f, prefix + "/" + name, out);
            else if (f.isFile()) out.add(new Item(f.getUri(), prefix + "/" + name));
        }
    }

    // ---- compartilhar ----
    @SuppressWarnings({"deprecation", "unchecked"})
    private void collectShare(Intent i) {
        if (i == null) return;
        String a = i.getAction();
        if (Intent.ACTION_SEND.equals(a)) {
            Uri u = i.getParcelableExtra(Intent.EXTRA_STREAM);
            if (u != null) shared.add(new Item(u, null));
        } else if (Intent.ACTION_SEND_MULTIPLE.equals(a)) {
            ArrayList<Uri> l = i.getParcelableArrayListExtra(Intent.EXTRA_STREAM);
            if (l != null) for (Uri u : l) shared.add(new Item(u, null));
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
            try {
                for (Item it : new ArrayList<>(shared)) {
                    Uri u = it.uri;
                    String name = u.getLastPathSegment();
                    long size = -1;
                    try (Cursor c = getContentResolver().query(u, null, null, null, null)) {
                        if (c != null && c.moveToFirst()) {
                            int n = c.getColumnIndex(OpenableColumns.DISPLAY_NAME), s = c.getColumnIndex(OpenableColumns.SIZE);
                            if (n >= 0) name = c.getString(n);
                            if (s >= 0 && !c.isNull(s)) size = c.getLong(s);
                        }
                    } catch (Exception ignored) { }
                    out.put(new JSONObject().put("name", name).put("size", size).put("rel", it.rel == null ? JSONObject.NULL : it.rel));
                }
            } catch (Exception ignored) { }
            return out.toString();
        }

        /** Abre o seletor de pastas do Android; o conteúdo (com subpastas) vira a lista de envio. */
        @JavascriptInterface
        public void pickFolder() {
            runOnUiThread(() -> folderPicker.launch(null));
        }

        @JavascriptInterface
        public void clearShare() {
            shared.clear();
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

        @JavascriptInterface
        public void changeServer() {
            runOnUiThread(() -> getBridge().getWebView().loadUrl(getBridge().getLocalUrl() + "/index.html?trocar=1"));
        }
    }
}
