export interface User {
  userId?: string;
  uniqueId?: string;
  nickname?: string;
  profilePictureUrl?: string;
}

export interface GiftData {
  giftId?: number;
  giftType?: number;
  repeatCount?: number;
  repeatEnd?: boolean;
  giftName?: string;
  diamondCount?: number;
  giftPictureUrl?: string;
  user?: User;
  receiverUserId?: string;
}

export interface RoomInfo {
  title?: string;
  owner?: {
    uniqueId?: string;
    nickname?: string;
    profilePictureUrl?: string;
  };
  userCount?: number;
  viewerCount?: number;
}

export type StatusState =
  | "idle"
  | "connecting"
  | "connected"
  | "disconnected"
  | "error"
  | "ended";

export interface StatusData {
  state: StatusState;
  username?: string;
  roomId?: string;
  roomInfo?: RoomInfo | null;
  code?: number;
  reason?: string;
  message?: string;
}

export type EventKind =
  | "status"
  | "chat"
  | "gift"
  | "member"
  | "like"
  | "follow"
  | "share"
  | "social"
  | "roomUser"
  | "emote"
  | "envelope"
  | "questionNew"
  | "liveIntro"
  | "linkMicBattle"
  | "linkMicArmies"
  | "subNotify";

export interface LiveEvent {
  type: EventKind;
  data: Record<string, unknown>;
  ts: number;
}
