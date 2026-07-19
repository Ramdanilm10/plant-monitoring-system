import {
  useEffect,
  useState,
} from "react";

import {
  getDSSAnalysis,
} from "../services/api";

function formatNumber(value) {
  const number = Number(value);

  if (!Number.isFinite(number)) {
    return "-";
  }

  return new Intl.NumberFormat(
    "id-ID",
    {
      maximumFractionDigits: 2,
    },
  ).format(number);
}

function formatDateTime(value) {
  if (!value) {
    return "-";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "-";
  }

  return `${new Intl.DateTimeFormat(
    "id-ID",
    {
      day: "2-digit",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
      timeZone: "Asia/Jakarta",
    },
  ).format(date)} WIB`;
}

function DSSAnalysisPanel({
  plantId,
  range,
}) {
  const [analysis, setAnalysis] =
    useState(null);

  const [isLoading, setIsLoading] =
    useState(false);

  const [errorMessage, setErrorMessage] =
    useState("");

  useEffect(() => {
    let isActive = true;

    async function loadAnalysis() {
      if (!plantId) {
        setAnalysis(null);
        setErrorMessage("");

        return;
      }

      setIsLoading(true);
      setErrorMessage("");

      try {
        const result =
          await getDSSAnalysis(
            plantId,
            range,
          );

        if (!isActive) {
          return;
        }

        setAnalysis(
          result?.data ?? result,
        );
      } catch (error) {
        if (!isActive) {
          return;
        }

        setAnalysis(null);

        setErrorMessage(
          error instanceof Error
            ? error.message
            : "Analisis DSS gagal dimuat.",
        );
      } finally {
        if (isActive) {
          setIsLoading(false);
        }
      }
    }

    loadAnalysis();

    return () => {
      isActive = false;
    };
  }, [
    plantId,
    range,
  ]);

  if (!plantId) {
    return null;
  }

  if (isLoading) {
    return (
      <section className="rounded-2xl border border-slate-200 bg-white p-6 text-center shadow-sm">
        <p className="text-sm text-slate-500">
          Menghitung analisis DSS...
        </p>
      </section>
    );
  }

  if (errorMessage) {
    return (
      <section className="rounded-2xl border border-red-200 bg-red-50 p-6">
        <p className="font-semibold text-red-800">
          Analisis DSS gagal dimuat
        </p>

        <p className="mt-2 text-sm text-red-700">
          {errorMessage}
        </p>
      </section>
    );
  }

  if (!analysis?.has_data) {
    return (
      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-sm font-semibold uppercase tracking-wider text-slate-500">
          Analisis DSS
        </p>

        <h2 className="mt-2 text-xl font-semibold text-slate-950">
          Data belum mencukupi
        </h2>

        <p className="mt-2 text-sm leading-6 text-slate-600">
          Belum tersedia data sensor
          pada rentang waktu yang dipilih.
        </p>
      </section>
    );
  }

  const metrics =
    analysis.metrics ?? {};

  const recommendations =
    Array.isArray(
      analysis.recommendations,
    )
      ? analysis.recommendations
      : [];

  return (
    <section className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <header className="border-b border-slate-100 p-6">
        <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <p className="text-sm font-semibold uppercase tracking-wider text-emerald-700">
              Decision Support System
            </p>

            <h2 className="mt-2 text-2xl font-semibold text-slate-950">
              Analisis kondisi historis
            </h2>

            <p className="mt-2 max-w-3xl text-sm leading-6 text-slate-600">
              {analysis.summary}
            </p>

            <div className="mt-4 rounded-xl bg-slate-50 px-4 py-3">
              <p className="text-xs font-semibold text-slate-600">
                Rentang analisis:{" "}
                {analysis.range_label} terakhir
              </p>

              <p className="mt-1 text-xs leading-5 text-slate-500">
                Kesimpulan analisis dihitung dari{" "}
                {analysis.total_readings} pembacaan
                yang tersedia pada rentang waktu
                yang dipilih.
              </p>

              {analysis.period_start &&
                analysis.period_end && (
                  <p className="mt-1 text-xs leading-5 text-slate-500">
                    Data pengukuran tersedia dari{" "}
                    {formatDateTime(
                      analysis.period_start,
                    )}{" "}
                    sampai{" "}
                    {formatDateTime(
                      analysis.period_end,
                    )}
                    .
                  </p>
                )}
            </div>
          </div>

          <div className="rounded-2xl bg-slate-950 px-6 py-5 text-white">
            <p className="text-xs font-semibold uppercase tracking-wider text-slate-300">
              Skor kesehatan
            </p>

            <p className="mt-2 text-4xl font-bold">
              {formatNumber(
                analysis.health_score,
              )}
            </p>

            <p className="mt-1 text-sm text-slate-300">
              dari 100
            </p>

            <span
              className={[
                "mt-4 inline-flex rounded-full px-3 py-1.5 text-xs font-semibold",
                getStatusClass(
                  analysis.status,
                ),
              ].join(" ")}
            >
              {analysis.status}
            </span>
          </div>
        </div>
      </header>

      <div className="p-6">
        <div className="grid gap-4 xl:grid-cols-3">
          <MetricCard
            metric={
              metrics.temperature
            }
            rangeLabel={
              analysis.range_label
            }
            totalReadings={
              analysis.total_readings
            }
          />

          <MetricCard
            metric={
              metrics.humidity
            }
            rangeLabel={
              analysis.range_label
            }
            totalReadings={
              analysis.total_readings
            }
          />

          <MetricCard
            metric={
              metrics.soil_moisture
            }
            rangeLabel={
              analysis.range_label
            }
            totalReadings={
              analysis.total_readings
            }
          />
        </div>

        <div className="mt-7">
          <h3 className="text-lg font-semibold text-slate-950">
            Rekomendasi perawatan
          </h3>

          <div className="mt-4 space-y-3">
            {recommendations.map(
              (
                recommendation,
                index,
              ) => (
                <RecommendationCard
                  key={`${recommendation.title}-${index}`}
                  recommendation={
                    recommendation
                  }
                />
              ),
            )}
          </div>
        </div>
      </div>
    </section>
  );
}

function MetricCard({
  metric,
  rangeLabel,
  totalReadings,
}) {
  if (!metric) {
    return null;
  }

  const periodLabel =
    rangeLabel || "rentang terpilih";

  return (
    <article className="rounded-2xl border border-slate-200 bg-slate-50 p-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h3 className="font-semibold text-slate-950">
            {metric.label}
          </h3>

          <p className="mt-1 text-xs text-slate-500">
            Batas ideal{" "}
            {formatNumber(
              metric.ideal_minimum,
            )}
            –{" "}
            {formatNumber(
              metric.ideal_maximum,
            )}
            {metric.unit}
          </p>
        </div>

        <span className="inline-flex w-fit rounded-full bg-white px-3 py-1.5 text-xs font-semibold text-slate-600 shadow-sm">
          Periode {periodLabel}
        </span>
      </div>

      <p className="mt-4 text-xs leading-5 text-slate-500">
        Nilai minimum, rata-rata, dan maksimum
        dihitung dari {totalReadings} pembacaan
        selama {periodLabel} terakhir.
      </p>

      <div className="mt-3 grid grid-cols-3 gap-2">
        <SmallStatistic
          label="Minimum"
          value={`${formatNumber(
            metric.minimum,
          )}${metric.unit}`}
          periodLabel={periodLabel}
        />

        <SmallStatistic
          label="Rata-rata"
          value={`${formatNumber(
            metric.average,
          )}${metric.unit}`}
          periodLabel={periodLabel}
        />

        <SmallStatistic
          label="Maksimum"
          value={`${formatNumber(
            metric.maximum,
          )}${metric.unit}`}
          periodLabel={periodLabel}
        />
      </div>

      <div className="mt-5 rounded-xl bg-white p-4">
        <p className="text-xs font-semibold text-slate-700">
          Distribusi kondisi selama{" "}
          {periodLabel} terakhir
        </p>

        <div className="mt-4 space-y-3">
          <PercentageBar
            label="Di bawah batas ideal"
            value={
              metric.below_percent
            }
            type="low"
          />

          <PercentageBar
            label="Dalam rentang ideal"
            value={
              metric.normal_percent
            }
            type="normal"
          />

          <PercentageBar
            label="Di atas batas ideal"
            value={
              metric.above_percent
            }
            type="high"
          />
        </div>

        <p className="mt-4 text-xs leading-5 text-slate-500">
          Ringkasan diatas menampilkan tiga indikator
          yang menunjukkan kondisi tanaman dalam
          rentang waktu yang dipilih.
        </p>
      </div>
    </article>
  );
}

function SmallStatistic({
  label,
  value,
  periodLabel,
}) {
  return (
    <div className="rounded-xl bg-white p-3">
      <p className="text-[11px] text-slate-500">
        {label}
      </p>

      <p className="mt-1 text-sm font-semibold text-slate-950">
        {value}
      </p>

      <p className="mt-1 text-[10px] leading-4 text-slate-400">
        Selama {periodLabel}
      </p>
    </div>
  );
}

function PercentageBar({
  label,
  value,
  type,
}) {
  const safeValue = Math.min(
    Math.max(
      Number(value) || 0,
      0,
    ),
    100,
  );

  const barClass = {
    low: "bg-amber-500",
    normal: "bg-emerald-600",
    high: "bg-red-500",
  }[type] ?? "bg-slate-500";

  return (
    <div>
      <div className="flex items-center justify-between gap-3">
        <p className="text-xs text-slate-600">
          {label}
        </p>

        <p className="text-xs font-semibold text-slate-800">
          {formatNumber(
            safeValue,
          )}
          %
        </p>
      </div>

      <div className="mt-1.5 h-2 overflow-hidden rounded-full bg-slate-200">
        <div
          className={[
            "h-full rounded-full",
            barClass,
          ].join(" ")}
          style={{
            width: `${safeValue}%`,
          }}
        />
      </div>
    </div>
  );
}

function RecommendationCard({
  recommendation,
}) {
  const className = {
    success:
      "border-emerald-200 bg-emerald-50 text-emerald-800",

    info:
      "border-blue-200 bg-blue-50 text-blue-800",

    warning:
      "border-amber-200 bg-amber-50 text-amber-800",

    danger:
      "border-red-200 bg-red-50 text-red-800",
  }[
    recommendation.level
  ] ??
    "border-slate-200 bg-slate-50 text-slate-800";

  return (
    <article
      className={[
        "rounded-xl border p-4",
        className,
      ].join(" ")}
    >
      <p className="font-semibold">
        {recommendation.title}
      </p>

      <p className="mt-1 text-sm leading-6 opacity-90">
        {recommendation.detail}
      </p>
    </article>
  );
}

function getStatusClass(status) {
  switch (status) {
    case "SANGAT BAIK":
      return "bg-emerald-500 text-white";

    case "BAIK":
      return "bg-blue-500 text-white";

    case "PERLU PERHATIAN":
      return "bg-amber-400 text-slate-950";

    case "KRITIS":
      return "bg-red-500 text-white";

    default:
      return "bg-slate-700 text-white";
  }
}

export default DSSAnalysisPanel;