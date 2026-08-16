import { createApp } from "vue";
import { VueQueryPlugin } from "@tanstack/vue-query";
import Sockudo from "@sockudo/client";
import { createSockudoPlugin } from "@sockudo/client/vue";
import { Toaster } from "vue-sonner";
import App from "./App.vue";
import router from "./router";
import "./style.css";

// Sockudo (Pusher-compatible) — client subscribe channel nhận events realtime.
// Go backend là publisher (HMAC) → channel "user_<username>" per TikTok user.
const sockudo = new Sockudo(import.meta.env.VITE_SOCKUDO_KEY ?? "demo-key", {
  wsHost: import.meta.env.VITE_SOCKUDO_HOST ?? "localhost",
  wsPort: Number(import.meta.env.VITE_SOCKUDO_PORT ?? 6002),
  forceTLS: false,
  enabledTransports: ["ws"],
});

const app = createApp(App);
app.use(router);
app.use(VueQueryPlugin);
app.use(createSockudoPlugin(sockudo));
app.component("Toaster", Toaster);
app.mount("#app");
