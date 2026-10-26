<script setup>
import { ref, computed, onMounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import Icon from "@/components/Icon.vue";
import * as authApi from "@/api/auth";
import { loadSession, ui, setUi, isDark } from "@/store/app";
import { cfg, appName, baseURL } from "@/lib/config";

const router = useRouter();
const route = useRoute();
const user = ref("");
const pass = ref("");
const otp = ref("");
const next1 = ref("");
const next2 = ref("");
const show = ref(false);
const error = ref("");
const busy = ref(false);
const step = ref("login"); // login | otp | change
const userInput = ref(null);

const oidc = computed(() => cfg.oidcAvailable);
onMounted(() => userInput.value?.focus());

function target() {
  const n = String(route.query.next || "");
  return n.startsWith("/") && !n.startsWith("//") && !n.startsWith("/login") ? n : "/";
}

async function finish() {
  const u = await loadSession();
  if (u) router.replace(target());
  else error.value = "Entrou, mas não foi possível carregar a sessão.";
}

async function submit() {
  error.value = "";
  if (!user.value || !pass.value) { error.value = "Informe usuário e senha."; return; }
  busy.value = true;
  try {
    if (step.value === "change") {
      if (next1.value.length < (cfg.passwordMinLength || 5)) throw new Error(`A nova senha precisa ter ao menos ${cfg.passwordMinLength || 5} caracteres.`);
      if (next1.value !== next2.value) throw new Error("As duas senhas novas não são iguais.");
      await authApi.changeRequiredPassword(user.value, pass.value, next1.value, otp.value);
      pass.value = next1.value;
      await authApi.login(user.value, pass.value, otp.value);
    } else {
      await authApi.login(user.value, pass.value, otp.value);
    }
    await finish();
  } catch (e) {
    if (e.code === "change-password") { step.value = "change"; error.value = ""; }
    else if (e.code === "otp") { step.value = "otp"; error.value = otp.value ? "Código incorreto." : ""; }
    else if (e.status === 401 || e.status === 403) error.value = step.value === "otp" ? "Código incorreto." : "Usuário ou senha incorretos.";
    else error.value = e.message || "Não foi possível entrar.";
  } finally {
    busy.value = false;
  }
}
const toggleTheme = () => setUi({ theme: isDark() ? "light" : "dark" });
</script>

<template>
  <section class="login" aria-label="Entrar">
    <svg class="lg-bg" viewBox="0 0 600 600" aria-hidden="true">
      <g fill="none" stroke-width="34">
        <circle cx="300" cy="300" r="250" stroke="var(--bar)" stroke-dasharray="52 8" opacity=".6" />
        <circle cx="300" cy="300" r="250" stroke="url(#gdB)" stroke-dasharray="52 8" stroke-dashoffset="-120" pathLength="1570" />
        <circle cx="300" cy="300" r="170" stroke="var(--bar)" opacity=".5" />
        <circle cx="300" cy="300" r="170" stroke="url(#gdA)" stroke-dasharray="620 450" />
      </g>
    </svg>
    <button class="ib lg-tema" type="button" aria-label="Alternar claro e escuro" title="Alternar claro e escuro" @click="toggleTheme"><Icon :name="isDark() ? 'sun' : 'moon'" /></button>
    <div class="lg-wrap">
      <form class="lg-card" novalidate @submit.prevent="submit">
        <div class="lg-brand"><span class="ico"><Icon name="hdd" /></span><span>{{ appName }}</span></div>
        <h1>{{ step === "change" ? "Escolha uma nova senha" : "Entrar" }}</h1>
        <p class="lg-sub">{{ step === "change" ? "Por segurança, troque a senha provisória agora." : step === "otp" ? "Digite o código do aplicativo autenticador." : "Acesse seus arquivos e pendrives." }}</p>
        <div v-if="error" class="lg-erro" role="alert"><Icon name="warning" /><span>{{ error }}</span></div>
        <label class="lg-campo"><Icon name="person" /><input ref="userInput" v-model="user" type="text" autocomplete="username" autocapitalize="off" spellcheck="false" placeholder="Usuário" aria-label="Usuário" :readonly="step !== 'login'" /></label>
        <label class="lg-campo"><Icon name="lock" /><input v-model="pass" :type="show ? 'text' : 'password'" autocomplete="current-password" :placeholder="step === 'change' ? 'Senha atual' : 'Senha'" aria-label="Senha" :readonly="step === 'otp'" />
          <button class="ib" type="button" :aria-label="show ? 'Ocultar senha' : 'Mostrar senha'" @click="show = !show"><Icon name="eye" /></button></label>
        <template v-if="step === 'change'">
          <label class="lg-campo"><Icon name="key" /><input v-model="next1" type="password" autocomplete="new-password" placeholder="Nova senha" aria-label="Nova senha" /></label>
          <label class="lg-campo"><Icon name="key" /><input v-model="next2" type="password" autocomplete="new-password" placeholder="Repita a nova senha" aria-label="Repita a nova senha" /></label>
        </template>
        <label v-if="step === 'otp'" class="lg-campo"><Icon name="shield" /><input v-model="otp" type="text" inputmode="numeric" autocomplete="one-time-code" placeholder="Código de 6 dígitos" aria-label="Código de verificação" autofocus /></label>
        <button class="lg-btn" type="submit" :disabled="busy">{{ busy ? "Aguarde…" : step === "change" ? "Salvar e entrar" : "Entrar" }}</button>
        <a v-if="oidc && step === 'login'" class="lg-alt" :href="baseURL + 'api/auth/oidc/login'">{{ cfg.oidcLoginButtonText || "Entrar com SSO" }}</a>
      </form>
      <div class="lg-rodape">{{ appName }} {{ cfg.version ? "· " + cfg.version : "" }} · base Filebrowser Quantum</div>
    </div>
  </section>
</template>
