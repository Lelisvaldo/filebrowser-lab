package io.github.lelisvaldo.filebrowserlab;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.Service;
import android.content.Context;
import android.content.Intent;
import android.content.pm.ServiceInfo;
import android.os.Build;
import android.os.IBinder;

import androidx.core.app.NotificationCompat;
import androidx.core.content.ContextCompat;

/**
 * Serviço em primeiro plano que mantém o app vivo enquanto há envios ou downloads em andamento
 * (o Android encerraria o processo em segundo plano). As notificações de progresso ficam por conta de quem chama.
 */
public class TransferService extends Service {
    public static final String CHANNEL = "transfers";
    public static final int FG_ID = 1;
    private static int active = 0;

    public static synchronized void begin(Context c) {
        if (active++ == 0) ContextCompat.startForegroundService(c, new Intent(c, TransferService.class));
    }

    public static synchronized void end(Context c) {
        if (--active <= 0) {
            active = 0;
            c.stopService(new Intent(c, TransferService.class));
        }
    }

    public static void ensureChannel(Context c) {
        if (Build.VERSION.SDK_INT < 26) return;
        NotificationManager nm = c.getSystemService(NotificationManager.class);
        if (nm.getNotificationChannel(CHANNEL) == null) {
            NotificationChannel ch = new NotificationChannel(CHANNEL, "Envios e downloads", NotificationManager.IMPORTANCE_LOW);
            ch.setDescription("Progresso dos envios e downloads do Filebrowser Lab");
            nm.createNotificationChannel(ch);
        }
    }

    public static PendingIntent openApp(Context c) {
        Intent i = new Intent(c, MainActivity.class).setFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP | Intent.FLAG_ACTIVITY_CLEAR_TOP);
        return PendingIntent.getActivity(c, 0, i, PendingIntent.FLAG_IMMUTABLE | PendingIntent.FLAG_UPDATE_CURRENT);
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        ensureChannel(this);
        Notification n = new NotificationCompat.Builder(this, CHANNEL)
                .setSmallIcon(android.R.drawable.stat_sys_upload)
                .setContentTitle("Filebrowser Lab")
                .setContentText("Transferindo arquivos…")
                .setOngoing(true)
                .setOnlyAlertOnce(true)
                .setContentIntent(openApp(this))
                .build();
        if (Build.VERSION.SDK_INT >= 29) startForeground(FG_ID, n, ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC);
        else startForeground(FG_ID, n);
        return START_NOT_STICKY;
    }

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }
}
