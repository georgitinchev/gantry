// A gantry crane in two strokes: dashed beam (desired), solid legs (actual).
export function Mark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true">
      <path d="M3 5h18" stroke="currentColor" strokeWidth="2" strokeDasharray="3 2" strokeLinecap="round" />
      <path d="M6 5v15M18 5v15" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      <rect x="9.5" y="11" width="5" height="5" rx="1" fill="var(--signal-fill)" stroke="currentColor" strokeWidth="1.5" />
      <path d="M12 5v6" stroke="currentColor" strokeWidth="1.5" />
    </svg>
  );
}
