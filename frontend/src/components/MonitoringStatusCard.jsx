const MONITORING_TIME_ZONE =
  "Asia/Jakarta";

function formatDateTime(value) {
  if (!value) {
    return "-";
  }

  const date = new Date(value);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return "-";
  }

  const formattedValue =
    new Intl.DateTimeFormat(
      "id-ID",
      {
        dateStyle: "medium",
        timeStyle: "medium",
        timeZone:
          MONITORING_TIME_ZONE,
      },
    ).format(date);

  return `${formattedValue} WIB`;
}

function formatAge(
  totalSeconds,
) {
  const seconds = Math.max(
    Number(totalSeconds) || 0,
    0,
  );

  if (seconds < 60) {
    return `${Math.floor(
      seconds,
    )} detik`;
  }

  const minutes = Math.floor(
    seconds / 60,
  );

  if (minutes < 60) {
    return `${minutes} menit`;
  }

  const hours = Math.floor(
    minutes / 60,
  );

  const remainingMinutes =
    minutes % 60;

  if (hours < 24) {
    if (
      remainingMinutes > 0
    ) {
      return `${hours} jam ${remainingMinutes} menit`;
    }

    return `${hours} jam`;
  }

  const days = Math.floor(
    hours / 24,
  );

  const remainingHours =
    hours % 24;

  if (remainingHours > 0) {
    return `${days} hari ${remainingHours} jam`;
  }

  return `${days} hari`;
}

function getDeviceStyle(
  status,
) {
  switch (
    String(
      status || "",
    ).toUpperCase()
  ) {
    case "ONLINE":
      return {
        label: "ONLINE",

        container:
          "border-emerald-200 bg-emerald-50",

        badge:
          "bg-emerald-100 text-emerald-700",

        dot:
          "bg-emerald-500",
      };

    case "OFFLINE":
      return {
        label: "OFFLINE",

        container:
          "border-red-200 bg-red-50",

        badge:
          "bg-red-100 text-red-700",

        dot:
          "bg-red-500",
      };

    default:
      return {
        label: "UNKNOWN",

        container:
          "border-amber-200 bg-amber-50",

        badge:
          "bg-amber-100 text-amber-700",

        dot:
          "bg-amber-500",
      };
  }
}

function getSyncStyle(
  status,
) {
  switch (
    String(
      status || "",
    ).toUpperCase()
  ) {
    case "CURRENT":
      return {
        label: "TERKINI",

        container:
          "border-emerald-200 bg-emerald-50",

        badge:
          "bg-emerald-100 text-emerald-700",

        dot:
          "bg-emerald-500",
      };

    case "DELAYED":
      return {
        label: "TERLAMBAT",

        container:
          "border-amber-200 bg-amber-50",

        badge:
          "bg-amber-100 text-amber-700",

        dot:
          "bg-amber-500",
      };

    default:
      return {
        label: "KEDALUWARSA",

        container:
          "border-red-200 bg-red-50",

        badge:
          "bg-red-100 text-red-700",

        dot:
          "bg-red-500",
      };
  }
}

function StatusBadge({
  label,
  badgeClass,
  dotClass,
}) {
  return (
    <span
      className={[
        "inline-flex items-center gap-2 rounded-full px-3 py-1.5 text-xs font-semibold",

        badgeClass,
      ].join(" ")}
    >
      <span
        className={[
          "h-2 w-2 rounded-full",

          dotClass,
        ].join(" ")}
      />

      {label}
    </span>
  );
}

function MonitoringStatusCard({
  monitoring,
  lastClientRefreshAt,
  isRefreshing,
}) {
  if (!monitoring) {
    return null;
  }

  const deviceStyle =
    getDeviceStyle(
      monitoring.device_status,
    );

  const syncStyle =
    getSyncStyle(
      monitoring.backend_sync_status,
    );

  return (
    <section>
      <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-semibold uppercase tracking-wider text-emerald-700">
            Status Monitoring
          </p>

          <h2 className="mt-1 text-xl font-semibold text-slate-950">
            Koneksi perangkat dan
            pembaruan data
          </h2>
        </div>

        <p className="text-xs text-slate-500">
          Refresh otomatis setiap{" "}
          {monitoring.auto_refresh_seconds ||
            30}{" "}
          detik
        </p>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <article
          className={[
            "rounded-2xl border p-5",

            deviceStyle.container,
          ].join(" ")}
        >
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="text-sm font-medium text-slate-600">
                Koneksi ESP32
              </p>

              <h3 className="mt-1 text-lg font-semibold text-slate-950">
                Status perangkat
              </h3>
            </div>

            <StatusBadge
              label={
                deviceStyle.label
              }
              badgeClass={
                deviceStyle.badge
              }
              dotClass={
                deviceStyle.dot
              }
            />
          </div>

          <p className="mt-4 text-sm leading-6 text-slate-600">
            {monitoring.device_message ||
              "Status perangkat belum tersedia."}
          </p>

          <div className="mt-4 border-t border-black/5 pt-4">
            <p className="text-xs text-slate-500">
              Pemeriksaan dilakukan
              melalui Blynk Cloud.
            </p>
          </div>
        </article>

        <article
          className={[
            "rounded-2xl border p-5",

            syncStyle.container,
          ].join(" ")}
        >
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="text-sm font-medium text-slate-600">
                Keterkinian data
              </p>

              <h3 className="mt-1 text-lg font-semibold text-slate-950">
                Data sensor terbaru
              </h3>
            </div>

            <StatusBadge
              label={
                syncStyle.label
              }
              badgeClass={
                syncStyle.badge
              }
              dotClass={
                syncStyle.dot
              }
            />
          </div>

          <p className="mt-4 text-sm leading-6 text-slate-600">
            {monitoring.backend_sync_message ||
              "Status data belum tersedia."}
          </p>

          <dl className="mt-4 space-y-2 border-t border-black/5 pt-4 text-xs">
            <div className="flex justify-between gap-4">
              <dt className="text-slate-500">
                Usia data
              </dt>

              <dd className="text-right font-semibold text-slate-700">
                {formatAge(
                  monitoring.data_age_seconds,
                )}
              </dd>
            </div>

            <div className="flex justify-between gap-4">
              <dt className="text-slate-500">
                Waktu pembacaan
              </dt>

              <dd className="text-right font-semibold text-slate-700">
                {formatDateTime(
                  monitoring.last_recorded_at,
                )}
              </dd>
            </div>

            <div className="flex justify-between gap-4">
              <dt className="text-slate-500">
                Sumber data
              </dt>

              <dd className="text-right font-semibold uppercase text-slate-700">
                {monitoring.source ||
                  "-"}
              </dd>
            </div>
          </dl>
        </article>

        <article className="rounded-2xl border border-sky-200 bg-sky-50 p-5">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="text-sm font-medium text-slate-600">
                Dashboard
              </p>

              <h3 className="mt-1 text-lg font-semibold text-slate-950">
                Auto-refresh
              </h3>
            </div>

            <StatusBadge
              label={
                isRefreshing
                  ? "MEMPERBARUI"
                  : "AKTIF"
              }
              badgeClass="bg-sky-100 text-sky-700"
              dotClass={
                isRefreshing
                  ? "animate-pulse bg-sky-500"
                  : "bg-sky-500"
              }
            />
          </div>

          <p className="mt-4 text-sm leading-6 text-slate-600">
            Nilai sensor dan status
            monitoring diperbarui
            otomatis tanpa memuat ulang
            halaman.
          </p>

          <dl className="mt-4 space-y-2 border-t border-black/5 pt-4 text-xs">
            <div className="flex justify-between gap-4">
              <dt className="text-slate-500">
                Refresh terakhir
              </dt>

              <dd className="text-right font-semibold text-slate-700">
                {formatDateTime(
                  lastClientRefreshAt,
                )}
              </dd>
            </div>

            <div className="flex justify-between gap-4">
              <dt className="text-slate-500">
                Interval
              </dt>

              <dd className="text-right font-semibold text-slate-700">
                {monitoring.auto_refresh_seconds ||
                  30}{" "}
                detik
              </dd>
            </div>
          </dl>
        </article>
      </div>
    </section>
  );
}

export default MonitoringStatusCard;