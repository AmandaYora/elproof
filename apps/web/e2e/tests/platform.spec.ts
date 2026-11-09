import { screenshotSuite } from "../lib/suite";
import { PLATFORM_DIALOGS, PLATFORM_PAGES } from "../lib/pages";

screenshotSuite("platform", PLATFORM_PAGES, PLATFORM_DIALOGS);
