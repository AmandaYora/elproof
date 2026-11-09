import { config } from "./lib/config";
import { stopStack } from "./lib/stack";

export default async function globalTeardown() {
  if (config.reuse) {
    console.log("[e2e] E2E_REUSE=1: leaving the stack up (stop it with `npm run e2e:down`)");
    return;
  }
  await stopStack({ dropDatabase: true });
}
