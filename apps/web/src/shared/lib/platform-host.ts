// ElProof's own hostnames. Any other hostname is a WO's (legacy) custom
// domain, where visitors are that WO's clients and must not see ElProof's
// own marketing site (ADR-0015; custom domains are being retired, K15).
const PLATFORM_HOSTNAMES = new Set(["elproof.elcodelabs.com", "localhost", "127.0.0.1"]);

export function isPlatformHost(hostname = window.location.hostname): boolean {
  return PLATFORM_HOSTNAMES.has(hostname);
}
