import { STORY_STEPS } from "@/modules/homepage/components/home/storySteps";

/** The "Cara kerja" steps as plain text: shown until the interactive story arrives. */
export function StoryStatic() {
  return (
    <ol className="mt-12 grid max-w-[640px] gap-8 border-l-2 border-border pl-6">
      {STORY_STEPS.map((step) => (
        <li key={step.when}>
          <p className="text-[14px] font-bold text-highlight-text">{step.when}</p>
          <h3 className="mt-1 text-[22px] leading-tight font-bold tracking-[-0.02em]">{step.title}</h3>
          <p className="mt-2 text-muted-foreground">{step.body}</p>
        </li>
      ))}
    </ol>
  );
}
