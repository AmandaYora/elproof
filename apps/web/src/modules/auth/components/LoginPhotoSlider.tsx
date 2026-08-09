import { useEffect, useState } from "react";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";

// Dwell time per slide before crossfading to the next one.
const SLIDE_DURATION_MS = 5000;

// Whole literal Tailwind tokens only (never string-concatenated fragments),
// same convention LoginPage.tsx's PANEL_CLASSES already uses -- Tailwind
// generates CSS by statically scanning source text for complete utility
// classes.
const OPACITY_CLASSES = { active: "opacity-100", inactive: "opacity-0" };

// Fetches the login page's fixed set of background photos as blob object
// URLs -- same responseType: "blob" + URL.createObjectURL pattern
// LoginPage.tsx's own useDomainBranding() already uses for a tenant logo.
// Requests fire in parallel; each slide is revealed as soon as its own blob
// resolves so the panel doesn't block on all of them before showing the
// first photo.
function useLoginSlides(): string[] {
  const [urls, setUrls] = useState<string[]>([]);

  useEffect(() => {
    let cancelled = false;
    const objectUrls: (string | undefined)[] = [];

    async function load() {
      try {
        const listRes = await httpClient.get(API.public.loginSlides);
        const count = (listRes.data.data as { count: number }).count;
        if (cancelled || count <= 0) return;

        for (let i = 0; i < count; i++) {
          httpClient
            .get(API.public.loginSlide(i), { responseType: "blob" })
            .then((res) => {
              if (cancelled) return;
              objectUrls[i] = URL.createObjectURL(res.data as Blob);
              setUrls(objectUrls.filter((u): u is string => u !== undefined));
            })
            .catch(() => {
              // One slide failing just means one fewer slide in rotation.
            });
        }
      } catch {
        // Endpoint unreachable/misconfigured -- LoginPhotoSlider renders
        // nothing, LoginPage's own gradient panel shows through instead.
      }
    }
    void load();

    return () => {
      cancelled = true;
      objectUrls.forEach((url) => url && URL.revokeObjectURL(url));
    };
  }, []);

  return urls;
}

// Full-bleed auto-advancing crossfade for LoginPage's left panel. Renders
// nothing until at least one photo has loaded, so the parent's existing
// gradient shows through as a loading/failure fallback. LoginPage.tsx itself
// still owns the dark gradient scrim and headline layered on top of this.
export function LoginPhotoSlider() {
  const urls = useLoginSlides();
  const [activeIndex, setActiveIndex] = useState(0);

  useEffect(() => {
    if (urls.length < 2) return;
    const timer = setInterval(() => {
      setActiveIndex((current) => (current + 1) % urls.length);
    }, SLIDE_DURATION_MS);
    return () => clearInterval(timer);
  }, [urls.length]);

  if (urls.length === 0) return null;
  const safeIndex = activeIndex % urls.length;

  return (
    <div className="absolute inset-0" aria-hidden="true">
      {urls.map((url, i) => (
        <img
          key={url}
          src={url}
          alt=""
          className={`absolute inset-0 h-full w-full object-cover transition-opacity duration-1000 ease-in-out ${
            i === safeIndex ? OPACITY_CLASSES.active : OPACITY_CLASSES.inactive
          }`}
        />
      ))}
    </div>
  );
}
