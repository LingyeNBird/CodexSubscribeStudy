import type { Component } from "vue";
import ClaudeCodeIcon from "./ClaudeCodeIcon.vue";
import OhMyPiIcon from "./OhMyPiIcon.vue";
import OpenCodeIcon from "./OpenCodeIcon.vue";
import CursorIcon from "./CursorIcon.vue";
import ClineIcon from "./ClineIcon.vue";
import RooCodeIcon from "./RooCodeIcon.vue";
import AiderIcon from "./AiderIcon.vue";

export const toolIcons: Record<string, Component> = {
  "Claude Code": ClaudeCodeIcon,
  "oh-my-pi": OhMyPiIcon,
  OpenCode: OpenCodeIcon,
  Cursor: CursorIcon,
  Cline: ClineIcon,
  "Roo Code": RooCodeIcon,
  Aider: AiderIcon,
};
