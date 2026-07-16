import { HelpFlow } from "@/components/help";

export function Header({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <div className="mb-6 flex items-start justify-between gap-4">
      <div>
        <h1 className="text-3xl font-semibold tracking-normal text-ink">{title}</h1>
        <p className="mt-1 max-w-3xl text-sm leading-6 text-ink/55">{subtitle}</p>
      </div>
      <HelpFlow page={title} />
    </div>
  );
}

export function Empty({ text }: { text: string }) {
  return <div className="rounded-md border border-dashed border-ink/15 p-4 text-sm text-ink/55">{text}</div>;
}

export function SkeletonBlock({ className = "h-24" }: { className?: string }) {
  return <div className={`skeleton-shimmer rounded-md ${className}`} />;
}

export function SkeletonText({ lines = 3 }: { lines?: number }) {
  return (
    <div className="grid gap-2">
      {Array.from({ length: lines }).map((_, index) => (
        <div key={index} className={`skeleton-shimmer h-3 rounded ${index === lines - 1 ? "w-2/3" : "w-full"}`} />
      ))}
    </div>
  );
}

export function PageLoading() {
  return (
    <main className="min-h-screen bg-[#f4f6f2] p-6 text-ink lg:pl-80">
      <div className="mx-auto max-w-7xl">
        <div className="mb-6">
          <SkeletonBlock className="h-10 max-w-sm" />
          <SkeletonBlock className="mt-3 h-4 max-w-2xl" />
        </div>
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
          {Array.from({ length: 5 }).map((_, index) => <SkeletonBlock key={index} className="h-28" />)}
        </div>
        <div className="mt-4 grid gap-4 xl:grid-cols-[1.1fr_.9fr]">
          <SkeletonBlock className="h-80" />
          <SkeletonBlock className="h-80" />
        </div>
        <div className="mt-4">
          <SkeletonBlock className="h-56" />
        </div>
      </div>
    </main>
  );
}
