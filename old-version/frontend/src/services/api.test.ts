import { describe, it, expect, beforeEach, afterEach } from "vitest";
import MockAdapter from "axios-mock-adapter";
import { api, fetchRoomPreview } from "./api";

let mock: MockAdapter;

beforeEach(() => {
  mock = new MockAdapter(api);
});

afterEach(() => {
  mock.restore();
});

describe("fetchRoomPreview (axios)", () => {
  it("returns live preview for a live user", async () => {
    mock.onGet("/api/room/mock.live").reply(200, {
      live: true,
      title: "Mock LIVE",
      userCount: 1234,
      owner: { uniqueId: "mock.live", nickname: "Mock Live" },
    });

    const data = await fetchRoomPreview("mock.live");
    expect(data.live).toBe(true);
    expect(data.title).toBe("Mock LIVE");
    expect(data.owner?.nickname).toBe("Mock Live");
  });

  it("returns live:false for an offline user", async () => {
    mock.onGet("/api/room/mock.offline").reply(200, { live: false });

    const data = await fetchRoomPreview("mock.offline");
    expect(data.live).toBe(false);
  });

  it("propagates network errors (thrown by axios)", async () => {
    mock.onGet("/api/room/error.user").networkError();

    await expect(fetchRoomPreview("error.user")).rejects.toThrow();
  });
});
