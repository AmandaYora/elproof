// `npm run e2e:stack` starts the stack and leaves it running; `npm run e2e:down` stops it.
import { startStack, stopStack } from "../lib/stack";

const cmd = process.argv[2] ?? "up";
if (cmd === "down") {
  await stopStack({ dropDatabase: true });
  console.log("[e2e] stack stopped");
} else {
  const state = await startStack();
  console.log(`[e2e] web ${state.webURL}  api ${state.apiURL}`);
  console.log("[e2e] sessions: e2e/.tmp/auth/*.json (owner, client, admin, empty)");
}
