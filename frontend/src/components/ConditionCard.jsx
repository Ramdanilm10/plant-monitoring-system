function ConditionCard({ condition }) {
  const isNormal =
    condition?.status?.toUpperCase() === "NORMAL";

  const statusClass = isNormal
    ? "bg-emerald-100 text-emerald-700"
    : "bg-amber-100 text-amber-800";

  const containerClass = isNormal
    ? "border-emerald-200 bg-emerald-50"
    : "border-amber-200 bg-amber-50";

  return (
    <section
      className={`rounded-2xl border p-6 ${containerClass}`}
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-slate-500">
            Kondisi saat ini
          </p>

          <h2 className="mt-1 text-xl font-semibold text-slate-900">
            {condition.message}
          </h2>
        </div>

        <span
          className={`rounded-full px-3 py-1 text-xs font-semibold tracking-wide ${statusClass}`}
        >
          {condition.status}
        </span>
      </div>

      <div className="mt-5 rounded-xl bg-white/70 p-4">
        <p className="text-xs font-semibold uppercase tracking-wider text-slate-500">
          Rekomendasi
        </p>

        <p className="mt-2 text-sm leading-6 text-slate-700">
          {condition.recommendation}
        </p>
      </div>
    </section>
  );
}

export default ConditionCard;