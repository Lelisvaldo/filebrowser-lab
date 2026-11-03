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
    private final List<Uri> shared = new ArrayList<>();
    private long pausedAt = 0;
    private boolean locked = false;
    private boolean uploading = false;

    @Override
    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        WebView wv = getBridge().getWebView();
        wv.addJavascriptInterface(new LabNative(), "LabNative");
        wv.setWebChromeClient(new BridgeWebChromeClient(getBridge()) {
            @Override
            public boolean onShowFileChooser(WebView view, ValueCallback<Uri[]> cb, FileChooserParams params) {
                if (!shared.isEmpty()) {
                    Uri[] arr = shared.toArray(new Uri[0]);
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
        if (Build.VERSION.SDK_INT >= 33 && checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED)
            ActivityCompat.requestPermissions(this, new String[]{Manifest.permission.POST_NOTIFICATIONS}, 7);
        applyBars(prefs().getBoolean("dark", true)); // última cor conhecida do site, para não piscar ao abrir
        collectShare(getIntent());
        lock();
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

    private void lock() {
        if (locked || !prefs().getBoolean("lock", true)) return;
        if (BiometricManager.from(this).canAuthenticate(authenticators()) != BiometricManager.BIOMETRIC_SUCCESS) return; // sem biometria/PIN configurado: não trava
        locked = true;
        final WebView wv = getBridge().getWebView();
        wv.setVisibility(View.INVISIBLE);
        BiometricPrompt prompt = new BiometricPrompt(this, ContextCompat.getMainExecutor(this), new BiometricPrompt.AuthenticationCallback() {
            @Override
            public void onAuthenticationSucceeded(BiometricPrompt.AuthenticationResult result) {
                locked = false;
                wv.setVisibility(View.VISIBLE);
            }

            @Override
            public void onAuthenticationError(int code, CharSequence msg) {
                // cancelou ou falhou: fecha o app em vez de deixar a tela aberta
                finishAndRemoveTask();
            }
        });
        BiometricPrompt.PromptInfo.Builder info = new BiometricPrompt.PromptInfo.Builder()
                .setTitle("Filebrowser Lab")
                .setSubtitle("Confirme que é você para abrir")
                .setAllowedAuthenticators(authenticators());
        if (Build.VERSION.SDK_INT < 30) info.setNegativeButtonText("Cancelar");
        prompt.authenticate(info.build());
    }

    // ---- compartilhar ----
    @SuppressWarnings({"deprecation", "unchecked"})
    private void collectShare(Intent i) {
        if (i == null) return;
        String a = i.getAction();
        if (Intent.ACTION_SEND.equals(a)) {
            Uri u = i.getParcelableExtra(Intent.EXTRA_STREAM);
            if (u != null) shared.add(u);
        } else if (Intent.ACTION_SEND_MULTIPLE.equals(a)) {
            ArrayList<Uri> l = i.getParcelableArrayListExtra(Intent.EXTRA_STREAM);
            if (l != null) shared.addAll(l);
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
                for (Uri u : new ArrayList<>(shared)) {
                    String name = u.getLastPathSegment();
                    long size = -1;
                    try (Cursor c = getContentResolver().query(u, null, null, null, null)) {
                        if (c != null && c.moveToFirst()) {
                            int n = c.getColumnIndex(OpenableColumns.DISPLAY_NAME), s = c.getColumnIndex(OpenableColumns.SIZE);
                            if (n >= 0) name = c.getString(n);
                            if (s >= 0 && !c.isNull(s)) size = c.getLong(s);
                        }
                    } catch (Exception ignored) { }
                    out.put(new JSONObject().put("name", name).put("size", size));
                }
            } catch (Exception ignored) { }
            return out.toString();
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

        @JavascriptInterface
        public void changeServer() {
            runOnUiThread(() -> getBridge().getWebView().loadUrl(getBridge().getLocalUrl() + "/index.html?trocar=1"));
        }
    }
}
