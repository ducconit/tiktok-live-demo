import { createRouter, createWebHistory } from "vue-router";
import { APP_TITLE } from "@/lib/app";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      name: "live",
      component: () => import("@/views/LiveView.vue"),
      meta: { title: "Theo dõi LIVE" },
    },
    { path: "/:pathMatch(.*)*", redirect: "/" },
  ],
});

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · ${APP_TITLE}` : APP_TITLE;
});

export default router;
