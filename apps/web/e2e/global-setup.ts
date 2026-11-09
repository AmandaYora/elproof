import { startStack } from "./lib/stack";

export default async function globalSetup() {
  await startStack();
}
