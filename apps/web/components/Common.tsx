export function Header({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <div className="mb-6">
      <h1 className="text-3xl font-semibold tracking-normal text-ink">{title}</h1>
      <p className="mt-1 max-w-3xl text-sm leading-6 text-ink/55">{subtitle}</p>
    </div>
  );
}

export function Empty({ text }: { text: string }) {
  return <div className="rounded-md border border-dashed border-ink/15 p-4 text-sm text-ink/55">{text}</div>;
}
