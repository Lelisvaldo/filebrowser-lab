import { createApp } from "vue";
import App from "./App.vue";
import { router } from "./router";
import { applyUi } from "./store/app";
import "./css/base.css";
import "./css/app.css";

applyUi();
createApp(App).use(router).mount("#app");
