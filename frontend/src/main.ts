import { createApp } from "vue";
import { VueQueryPlugin } from "@tanstack/vue-query";
import Sockudo from "@sockudo/client";
import { createSockudoPlugin } from "@sockudo/client/vue";
import App from "./App.vue";
import "./style.css";

// Sockudo (Pusher-compatible) — client subscribe channel nhận events realtime.
const sockudo = new Sockudo(import.meta.env.VITE_SOCKUDO_KEY ?? "demo-key", {
  wsHost: import.meta.env.VITE_SOCKUDO_HOST ?? "localhost",
  wsPort: Number(import.meta.env.VITE_SOCKUDO_PORT ?? 6001),
  forceTLS: false,
  enabledTransports: ["ws"],
});

const app = createApp(App);
app.use(VueQueryPlugin);
app.use(createSockudoPlugin(sockudo));
app.mount("#app");
