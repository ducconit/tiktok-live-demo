import { describe, it, expect, vi } from "vitest";
import { mount } from "@vue/test-utils";
import ConnectBar from "./ConnectBar.vue";

vi.mock("@/composables/useRoomQuery", () => ({
  useRoomPreview: () => ({ data: { value: null }, isPending: { value: false } }),
}));

describe("ConnectBar", () => {
  it("emits connect with the trimmed username", async () => {
    const wrapper = mount(ConnectBar, {
      props: { connected: false, connecting: false },
    });
    await wrapper.find("input").setValue("  nhu2hand2  ");
    await wrapper.find("button").trigger("click");
    expect(wrapper.emitted("connect")).toEqual([["nhu2hand2"]]);
  });

  it("shows Dừng when connected and emits disconnect without re-connecting", async () => {
    const wrapper = mount(ConnectBar, {
      props: { connected: true, connecting: false },
    });
    expect(wrapper.text()).toContain("Dừng");
    await wrapper.find("button").trigger("click");
    expect(wrapper.emitted("disconnect")).toHaveLength(1);
    // The Dừng button must NOT trigger a connect (the auto-reconnect bug).
    expect(wrapper.emitted("connect")).toBeUndefined();
  });

  it("disables Kết nối while connecting", () => {
    const wrapper = mount(ConnectBar, {
      props: { connected: false, connecting: true },
    });
    expect(wrapper.text()).toContain("Đang kết nối…");
    const button = wrapper.find("button");
    expect(button.attributes("disabled")).toBeDefined();
  });
});
