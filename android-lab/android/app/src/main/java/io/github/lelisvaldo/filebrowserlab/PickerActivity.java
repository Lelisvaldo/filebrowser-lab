package io.github.lelisvaldo.filebrowserlab;

import android.Manifest;
import android.app.Activity;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.graphics.Typeface;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.os.Environment;
import android.provider.Settings;
import android.text.format.Formatter;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.view.Window;
import android.widget.BaseAdapter;
import android.widget.Button;
import android.widget.CheckBox;
import android.widget.LinearLayout;
import android.widget.ListView;
import android.widget.TextView;

import java.io.File;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Comparator;
import java.util.LinkedHashSet;
import java.util.Set;

/**
 * Seletor do app: mostra as pastas e os arquivos do armazenamento do aparelho numa lista só.
 * Cada item tem uma caixa; marcar uma pasta leva tudo que está dentro dela. O usuário não precisa
 * decidir "arquivo ou pasta": o app entrega o que foi marcado e o envio recria a estrutura no servidor.
 * Precisa da permissão "Acesso a todos os arquivos" (Android 11+) ou de leitura do armazenamento (antes disso).
 */
public class PickerActivity extends Activity {
    public static final String EXTRA_PATHS = "paths";

    private static final int BG = 0xFF170A38, BAR = 0xFF120A33, FG = 0xFFEDEBF7, MUTED = 0xFFA79FC9, ACC = 0xFF8B5CF6;

    private final Set<String> picked = new LinkedHashSet<>();
    private File root;
    private File cur;
    private File[] kids = new File[0];
    private boolean denied = false; // pasta que o Android não deixa listar

    private LinearLayout page;
    private TextView path;
    private TextView hint;
    private ListView list;
    private Button send;
    private Button up;
    private Adapter adapter;

    @Override
    protected void onCreate(Bundle b) {
        super.onCreate(b);
        Window w = getWindow();
        w.setStatusBarColor(BAR);
        w.setNavigationBarColor(BAR);
        root = Environment.getExternalStorageDirectory();
        cur = root;
        if (b != null && b.getStringArrayList("picked") != null) picked.addAll(b.getStringArrayList("picked"));
        build();
    }

    @Override
    protected void onSaveInstanceState(Bundle o) {
        super.onSaveInstanceState(o);
        o.putStringArrayList("picked", new ArrayList<>(picked));
    }

    @Override
    protected void onResume() {
        super.onResume();
        if (!allowed()) { showPermission(); return; }
        showList();
    }

    @Override
    public void onBackPressed() {
        if (allowed() && !cur.equals(root)) { open(cur.getParentFile()); return; }
        setResult(RESULT_CANCELED);
        finish();
    }

    // ---- permissão ----
    private boolean allowed() {
        if (Build.VERSION.SDK_INT >= 30) return Environment.isExternalStorageManager();
        return checkSelfPermission(Manifest.permission.READ_EXTERNAL_STORAGE) == PackageManager.PERMISSION_GRANTED;
    }

    private void askPermission() {
        if (Build.VERSION.SDK_INT >= 30) {
            Intent i = new Intent(Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION, Uri.parse("package:" + getPackageName()));
            try { startActivity(i); } catch (Exception e) { startActivity(new Intent(Settings.ACTION_MANAGE_ALL_FILES_ACCESS_PERMISSION)); }
        } else {
            requestPermissions(new String[]{Manifest.permission.READ_EXTERNAL_STORAGE}, 11);
        }
    }

    @Override
    public void onRequestPermissionsResult(int code, String[] perms, int[] res) {
        super.onRequestPermissionsResult(code, perms, res);
        if (allowed()) showList(); else showPermission();
    }

    // ---- telas ----
    private int dp(int v) { return (int) (v * getResources().getDisplayMetrics().density + 0.5f); }

    private TextView text(String s, int sp, int color, boolean bold) {
        TextView t = new TextView(this);
        t.setText(s);
        t.setTextSize(sp);
        t.setTextColor(color);
        if (bold) t.setTypeface(Typeface.DEFAULT_BOLD);
        return t;
    }

    private Button button(String s, boolean primary) {
        Button b = new Button(this);
        b.setText(s);
        b.setAllCaps(false);
        b.setTextColor(0xFFFFFFFF);
        b.setBackgroundColor(primary ? ACC : 0xFF2A1B5E);
        return b;
    }

    private void build() {
        page = new LinearLayout(this);
        page.setOrientation(LinearLayout.VERTICAL);
        page.setBackgroundColor(BG);
        setContentView(page);
    }

    private void showPermission() {
        page.removeAllViews();
        LinearLayout box = new LinearLayout(this);
        box.setOrientation(LinearLayout.VERTICAL);
        box.setGravity(Gravity.CENTER_VERTICAL);
        box.setPadding(dp(24), dp(24), dp(24), dp(24));
        box.addView(text("Permitir o acesso aos arquivos", 22, FG, true));
        TextView why = text("Para você escolher pastas e arquivos do celular e enviá-los ao servidor, o Android exige a permissão "
                + "\"Acesso a todos os arquivos\". O app só lê o que você marcar para enviar; nada é enviado sem o seu toque, e nada sai da sua rede.\n\n"
                + "Na próxima tela, ligue a chave do Filebrowser Lab e volte. Se você não permitir, não dá para escolher arquivos do celular por aqui.", 15, MUTED, false);
        why.setPadding(0, dp(12), 0, dp(20));
        box.addView(why);
        Button ok = button("Permitir", true);
        ok.setOnClickListener(v -> askPermission());
        box.addView(ok);
        Button no = button("Agora não", false);
        no.setOnClickListener(v -> { setResult(RESULT_CANCELED); finish(); });
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        lp.topMargin = dp(10);
        box.addView(no, lp);
        page.addView(box, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f));
        list = null;
    }

    private void showList() {
        if (list != null) { refresh(); return; }
        page.removeAllViews();

        LinearLayout head = new LinearLayout(this);
        head.setOrientation(LinearLayout.HORIZONTAL);
        head.setGravity(Gravity.CENTER_VERTICAL);
        head.setBackgroundColor(BAR);
        head.setPadding(dp(8), dp(8), dp(12), dp(8));
        up = button("‹", false);
        up.setOnClickListener(v -> { if (!cur.equals(root)) open(cur.getParentFile()); });
        head.addView(up, new LinearLayout.LayoutParams(dp(48), dp(44)));
        path = text("", 15, FG, true);
        path.setPadding(dp(12), 0, 0, 0);
        path.setSingleLine(true);
        path.setEllipsize(android.text.TextUtils.TruncateAt.START);
        head.addView(path, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        page.addView(head, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));

        hint = text("", 14, MUTED, false);
        hint.setPadding(dp(16), dp(10), dp(16), dp(10));
        page.addView(hint);

        list = new ListView(this);
        list.setDividerHeight(0);
        adapter = new Adapter();
        list.setAdapter(adapter);
        list.setOnItemClickListener((p, v, pos, id) -> {
            File f = kids[pos];
            if (f.isDirectory()) open(f); else toggle(f);
        });
        page.addView(list, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f));

        LinearLayout foot = new LinearLayout(this);
        foot.setOrientation(LinearLayout.HORIZONTAL);
        foot.setBackgroundColor(BAR);
        foot.setPadding(dp(12), dp(10), dp(12), dp(10));
        Button cancel = button("Cancelar", false);
        cancel.setOnClickListener(v -> { setResult(RESULT_CANCELED); finish(); });
        foot.addView(cancel, new LinearLayout.LayoutParams(0, dp(48), 1f));
        send = button("Enviar", true);
        send.setOnClickListener(v -> done());
        LinearLayout.LayoutParams sp = new LinearLayout.LayoutParams(0, dp(48), 2f);
        sp.leftMargin = dp(10);
        foot.addView(send, sp);
        page.addView(foot, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        refresh();
    }

    private void open(File dir) {
        if (dir == null) return;
        cur = dir;
        refresh();
    }

    private void refresh() {
        File[] l = cur.listFiles();
        denied = l == null;
        kids = l == null ? new File[0] : l;
        Arrays.sort(kids, new Comparator<File>() {
            @Override
            public int compare(File a, File b) {
                if (a.isDirectory() != b.isDirectory()) return a.isDirectory() ? -1 : 1;
                return a.getName().compareToIgnoreCase(b.getName());
            }
        });
        // esconde itens com nome começando por ponto (cache, miniaturas, lixeiras do sistema)
        ArrayList<File> vis = new ArrayList<>();
        for (File f : kids) if (!f.getName().startsWith(".")) vis.add(f);
        kids = vis.toArray(new File[0]);
        String rel = cur.equals(root) ? "Armazenamento interno" : cur.getAbsolutePath().substring(root.getAbsolutePath().length());
        path.setText(rel);
        up.setEnabled(!cur.equals(root));
        up.setAlpha(cur.equals(root) ? 0.3f : 1f);
        hint.setVisibility(View.VISIBLE);
        if (denied) hint.setText("O Android não deixa o app ver esta pasta. Para as pastas de outros apps (como Android/data), use o ADB (veja o guia).");
        else if (kids.length == 0) hint.setText("Pasta vazia. Você pode marcá-la na pasta de cima; ela será recriada no servidor.");
        else hint.setText("Toque numa pasta para abrir; use a caixa para marcar pastas e arquivos.");
        adapter.notifyDataSetChanged();
        list.setSelection(0);
        sync();
    }

    private void toggle(File f) {
        String p = f.getAbsolutePath();
        if (!picked.remove(p)) picked.add(p);
        adapter.notifyDataSetChanged();
        sync();
    }

    private boolean covered(File f) {
        // marcado ele mesmo ou dentro de uma pasta marcada
        File x = f;
        while (x != null) {
            if (picked.contains(x.getAbsolutePath())) return true;
            if (x.equals(root)) break;
            x = x.getParentFile();
        }
        return false;
    }

    private void sync() {
        int n = picked.size();
        send.setText(n == 0 ? "Marque o que enviar" : "Enviar (" + n + ")");
        send.setEnabled(n > 0);
        send.setAlpha(n > 0 ? 1f : 0.45f);
    }

    private void done() {
        if (picked.isEmpty()) return;
        Intent r = new Intent();
        r.putStringArrayListExtra(EXTRA_PATHS, new ArrayList<>(picked));
        setResult(RESULT_OK, r);
        finish();
    }

    private final class Adapter extends BaseAdapter {
        @Override public int getCount() { return kids.length; }
        @Override public Object getItem(int i) { return kids[i]; }
        @Override public long getItemId(int i) { return i; }

        @Override
        public View getView(int i, View old, ViewGroup parent) {
            LinearLayout row = new LinearLayout(PickerActivity.this);
            row.setOrientation(LinearLayout.HORIZONTAL);
            row.setGravity(Gravity.CENTER_VERTICAL);
            row.setPadding(dp(8), dp(4), dp(16), dp(4));
            final File f = kids[i];
            CheckBox cb = new CheckBox(PickerActivity.this);
            boolean inherited = covered(f) && !picked.contains(f.getAbsolutePath());
            cb.setChecked(covered(f));
            cb.setEnabled(!inherited); // dentro de uma pasta já marcada: já está incluído
            cb.setButtonTintList(android.content.res.ColorStateList.valueOf(ACC));
            cb.setOnClickListener(v -> toggle(f));
            row.addView(cb, new LinearLayout.LayoutParams(dp(52), dp(52)));
            LinearLayout col = new LinearLayout(PickerActivity.this);
            col.setOrientation(LinearLayout.VERTICAL);
            col.addView(text((f.isDirectory() ? "📁  " : "") + f.getName(), 16, FG, f.isDirectory()));
            String sub = f.isDirectory() ? "Pasta" : Formatter.formatFileSize(PickerActivity.this, f.length());
            col.addView(text(sub, 12, MUTED, false));
            row.addView(col, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
            if (f.isDirectory()) row.addView(text("›", 22, MUTED, false));
            return row;
        }
    }
}
