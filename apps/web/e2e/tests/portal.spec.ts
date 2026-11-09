import { screenshotSuite } from "../lib/suite";
import { PORTAL_PAGES } from "../lib/pages";

// The Client Portal is phone-first; it is shot on both viewports. No dialog: its only overlay is the
// evidence viewer, which needs object storage (E2E_S3_ENDPOINT) and is not part of the baseline.
screenshotSuite("portal", PORTAL_PAGES);
