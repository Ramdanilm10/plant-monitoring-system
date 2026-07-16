function SensorCard({ title, value, unit, description }) {
  return (
    <article className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <p className="text-sm font-medium text-slate-500">
        {title}
      </p>

      <div className="mt-3 flex items-end gap-1">
        <span className="text-3xl font-semibold tracking-tight text-slate-900">
          {value}
        </span>

        <span className="pb-1 text-sm font-medium text-slate-500">
          {unit}
        </span>
      </div>

      <p className="mt-3 text-sm leading-6 text-slate-500">
        {description}
      </p>
    </article>
  );
}

export default SensorCard;