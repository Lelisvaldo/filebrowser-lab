package io.github.lelisvaldo.filebrowserlab;

import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.ContentResolver;
import android.content.ContentValues;
import android.content.Context;
import android.content.Intent;
import android.net.Uri;
import android.os.Build;
import android.os.Environment;
import android.provider.MediaStore;
import android.webkit.CookieManager;
import android.webkit.URLUtil;

import androidx.core.app.NotificationCompat;
import androidx.core.content.FileProvider;

import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.concurrent.atomic.AtomicInteger;

/**
 * Baixa pelo próprio app (e não pelo DownloadManager do sistema) para valer o certificado da casa instalado no aparelho.
 * Usa o cookie de sessão do WebView, grava em Downloads e avisa com uma notificação que abre o arquivo.
 */
public final class Downloader {
    private static final AtomicInteger IDS = new AtomicInteger(100);

    private Downloader() { }

    public static void start(final Context ctx, final String url, final String userAgent, final String contentDisposition, final String mime) {
        final int id = IDS.incrementAndGet();
        final String name = URLUtil.guessFileName(url, contentDisposition, mime);
        new Thread(() -> run(ctx.getApplicationContext(), id, url, userAgent, name, mime), "lab-download").start();
    }

    private static void run(Context c, int id, String url, String ua, String name, String mime) {
        TransferService.ensureChannel(c);
        NotificationManager nm = c.getSystemService(NotificationManager.class);
        TransferService.begin(c);
        NotificationCompat.Builder b = new NotificationCompat.Builder(c, TransferService.CHANNEL)
                .setSmallIcon(android.R.drawable.stat_sys_download).setContentTitle(name).setContentText("Baixando…")
                .setOngoing(true).setOnlyAlertOnce(true).setProgress(0, 0, true).setContentIntent(TransferService.openApp(c));
        try { nm.notify(id, b.build()); } catch (SecurityException ignored) { }
        Uri result = null;
        String error = null;
        HttpURLConnection con = null;
        try {
            con = (HttpURLConnection) new URL(url).openConnection();
            String cookie = CookieManager.getInstance().getCookie(url);
            if (cookie != null) con.setRequestProperty("Cookie", cookie);
            if (ua != null) con.setRequestProperty("User-Agent", ua);
            con.setConnectTimeout(15000);
            con.setReadTimeout(60000);
            if (con.getResponseCode() != 200) throw new Exception("o servidor respondeu " + con.getResponseCode());
            long total = con.getContentLengthLong();
            String type = mime != null && !mime.isEmpty() ? mime : "application/octet-stream";
            OutputStream out;
            Uri uri = null;
            File file = null;
            if (Build.VERSION.SDK_INT >= 29) {
                ContentValues v = new ContentValues();
                v.put(MediaStore.Downloads.DISPLAY_NAME, name);
                v.put(MediaStore.Downloads.MIME_TYPE, type);
                v.put(MediaStore.Downloads.RELATIVE_PATH, Environment.DIRECTORY_DOWNLOADS);
                v.put(MediaStore.Downloads.IS_PENDING, 1);
                ContentResolver r = c.getContentResolver();
                uri = r.insert(MediaStore.Downloads.EXTERNAL_CONTENT_URI, v);
                if (uri == null) throw new Exception("não consegui criar o arquivo em Downloads");
                out = r.openOutputStream(uri);
            } else {
                File dir = c.getExternalFilesDir(Environment.DIRECTORY_DOWNLOADS);
                file = new File(dir, name);
                out = new FileOutputStream(file);
            }
            long done = 0;
            int lastPct = -1;
            try (InputStream in = con.getInputStream(); OutputStream o = out) {
                byte[] buf = new byte[64 * 1024];
                int n;
                while ((n = in.read(buf)) > 0) {
                    o.write(buf, 0, n);
                    done += n;
                    if (total > 0) {
                        int pct = (int) (done * 100 / total);
                        if (pct != lastPct) {
                            lastPct = pct;
                            b.setProgress(100, pct, false).setContentText("Baixando… " + pct + "%");
                            try { nm.notify(id, b.build()); } catch (SecurityException ignored) { }
                        }
                    }
                }
            }
            if (uri != null) {
                ContentValues v = new ContentValues();
                v.put(MediaStore.Downloads.IS_PENDING, 0);
                c.getContentResolver().update(uri, v, null, null);
                result = uri;
            } else {
                result = FileProvider.getUriForFile(c, c.getPackageName() + ".fileprovider", file);
            }
            Intent view = new Intent(Intent.ACTION_VIEW).setDataAndType(result, type).addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
            PendingIntent pi = PendingIntent.getActivity(c, id, Intent.createChooser(view, "Abrir com").addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION),
                    PendingIntent.FLAG_IMMUTABLE | PendingIntent.FLAG_UPDATE_CURRENT);
            NotificationCompat.Builder ok = new NotificationCompat.Builder(c, TransferService.CHANNEL)
                    .setSmallIcon(android.R.drawable.stat_sys_download_done).setContentTitle(name).setContentText("Baixado em Downloads. Toque para abrir.")
                    .setAutoCancel(true).setContentIntent(pi);
            try { nm.notify(id, ok.build()); } catch (SecurityException ignored) { }
        } catch (Exception e) {
            error = e.getMessage() == null ? e.toString() : e.getMessage();
            NotificationCompat.Builder bad = new NotificationCompat.Builder(c, TransferService.CHANNEL)
                    .setSmallIcon(android.R.drawable.stat_notify_error).setContentTitle(name).setContentText("Não baixou: " + error)
                    .setAutoCancel(true).setContentIntent(TransferService.openApp(c));
            try { nm.notify(id, bad.build()); } catch (SecurityException ignored) { }
        } finally {
            if (con != null) con.disconnect();
            TransferService.end(c);
        }
    }
}
