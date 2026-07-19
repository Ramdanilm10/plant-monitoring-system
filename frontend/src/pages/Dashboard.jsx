import {
  useCallback,
  useEffect,
  useState,
} from "react";

import ConditionCard from "../components/ConditionCard";
import DSSAnalysisPanel from "../components/DSSAnalysisPanel";
import HistoryExportButtons from "../components/HistoryExportButtons";
import MonitoringStatusCard from "../components/MonitoringStatusCard";
import SensorCard from "../components/SensorCard";

import SensorHistoryChart, {
  CombinedSensorHistoryChart,
} from "../components/SensorHistoryChart";

import {
  getAvailablePlants,
  getDashboard,
  getSensorHistory,
} from "../services/api";

const DASHBOARD_REFRESH_INTERVAL_MS = 30_000;

const HISTORY_RANGES = [
  {
    value: "1h",
    label: "1 Jam",
  },
  {
    value: "6h",
    label: "6 Jam",
  },
  {
    value: "12h",
    label: "12 Jam",
  },
  {
    value: "24h",
    label: "24 Jam",
  },
  {
    value: "7d",
    label: "7 Hari",
  },
  {
    value: "30d",
    label: "30 Hari",
  },
];

function Dashboard() {
  const [plants, setPlants] = useState([]);

  const [
    selectedPlantId,
    setSelectedPlantId,
  ] = useState(null);

  const [
    selectedRange,
    setSelectedRange,
  ] = useState("24h");

  const [dashboard, setDashboard] =
    useState(null);

  const [history, setHistory] =
    useState(null);

  const [
    lastClientRefreshAt,
    setLastClientRefreshAt,
  ] = useState(null);

  const [
    isLoadingPlants,
    setIsLoadingPlants,
  ] = useState(true);

  const [
    isLoadingDashboard,
    setIsLoadingDashboard,
  ] = useState(false);

  const [
    isAutoRefreshing,
    setIsAutoRefreshing,
  ] = useState(false);

  const [
    isLoadingHistory,
    setIsLoadingHistory,
  ] = useState(false);

  const [
    plantListError,
    setPlantListError,
  ] = useState("");

  const [
    dashboardError,
    setDashboardError,
  ] = useState("");

  const [
    refreshWarning,
    setRefreshWarning,
  ] = useState("");

  const [
    historyError,
    setHistoryError,
  ] = useState("");

  const loadPlants = useCallback(async () => {
    setIsLoadingPlants(true);
    setPlantListError("");

    try {
      const result =
        await getAvailablePlants();

      const plantList =
        Array.isArray(result)
          ? result
          : result?.data;

      const normalizedPlants =
        Array.isArray(plantList)
          ? plantList
          : [];

      setPlants(normalizedPlants);

      setSelectedPlantId(
        (currentPlantId) => {
          const stillExists =
            normalizedPlants.some(
              (plant) =>
                String(plant.id) ===
                String(currentPlantId),
            );

          if (stillExists) {
            return currentPlantId;
          }

          return (
            normalizedPlants[0]?.id ??
            null
          );
        },
      );

      if (
        normalizedPlants.length === 0
      ) {
        setDashboard(null);
        setHistory(null);
      }
    } catch (error) {
      setPlants([]);
      setSelectedPlantId(null);
      setDashboard(null);
      setHistory(null);

      setPlantListError(
        error instanceof Error
          ? error.message
          : "Daftar tanaman gagal dimuat.",
      );
    } finally {
      setIsLoadingPlants(false);
    }
  }, []);

  const loadDashboard = useCallback(
    async (
      plantId,
      {
        silent = false,
      } = {},
    ) => {
      if (!plantId) {
        setDashboard(null);
        setDashboardError("");
        setRefreshWarning("");

        return;
      }

      if (silent) {
        setIsAutoRefreshing(true);
      } else {
        setIsLoadingDashboard(true);
        setDashboardError("");
      }

      try {
        const result =
          await getDashboard(plantId);

        setDashboard(
          result?.data ?? result,
        );

        setDashboardError("");
        setRefreshWarning("");

        setLastClientRefreshAt(
          new Date(),
        );
      } catch (error) {
        const message =
          error instanceof Error
            ? error.message
            : "Data dashboard gagal dimuat.";

        if (silent) {
          setRefreshWarning(
            `Pembaruan otomatis gagal: ${message}`,
          );
        } else {
          setDashboard(null);
          setDashboardError(message);
        }
      } finally {
        if (silent) {
          setIsAutoRefreshing(false);
        } else {
          setIsLoadingDashboard(false);
        }
      }
    },
    [],
  );

  const loadHistory = useCallback(
    async (plantId, range) => {
      if (!plantId) {
        setHistory(null);
        setHistoryError("");

        return;
      }

      setIsLoadingHistory(true);
      setHistoryError("");

      try {
        const result =
          await getSensorHistory(
            plantId,
            range,
          );

        setHistory(
          result?.data ?? result,
        );
      } catch (error) {
        setHistory(null);

        setHistoryError(
          error instanceof Error
            ? error.message
            : "Histori sensor gagal dimuat.",
        );
      } finally {
        setIsLoadingHistory(false);
      }
    },
    [],
  );

  useEffect(() => {
    loadPlants();
  }, [loadPlants]);

  useEffect(() => {
    loadDashboard(selectedPlantId);
  }, [
    selectedPlantId,
    loadDashboard,
  ]);

  useEffect(() => {
    loadHistory(
      selectedPlantId,
      selectedRange,
    );
  }, [
    selectedPlantId,
    selectedRange,
    loadHistory,
  ]);

  useEffect(() => {
    if (!selectedPlantId) {
      return undefined;
    }

    function refreshDashboard() {
      if (
        document.visibilityState !==
        "visible"
      ) {
        return;
      }

      loadDashboard(
        selectedPlantId,
        {
          silent: true,
        },
      );
    }

    const intervalId =
      window.setInterval(
        refreshDashboard,
        DASHBOARD_REFRESH_INTERVAL_MS,
      );

    function handleVisibilityChange() {
      if (
        document.visibilityState ===
        "visible"
      ) {
        refreshDashboard();
      }
    }

    document.addEventListener(
      "visibilitychange",
      handleVisibilityChange,
    );

    return () => {
      window.clearInterval(
        intervalId,
      );

      document.removeEventListener(
        "visibilitychange",
        handleVisibilityChange,
      );
    };
  }, [
    selectedPlantId,
    loadDashboard,
  ]);

  async function handleReload() {
    setRefreshWarning("");

    await loadPlants();

    if (!selectedPlantId) {
      return;
    }

    await Promise.all([
      loadDashboard(
        selectedPlantId,
      ),

      loadHistory(
        selectedPlantId,
        selectedRange,
      ),
    ]);
  }

  const isReloading =
    isLoadingPlants ||
    isLoadingDashboard ||
    isLoadingHistory;

  const hasSensorData =
    dashboard?.has_sensor_data ===
    true;

  const hasNoSensorData =
    dashboard?.has_sensor_data ===
    false;

  const historyReadings =
    Array.isArray(
      history?.readings,
    )
      ? history.readings
      : [];

  const selectedPlant =
    plants.find(
      (plant) =>
        String(plant.id) ===
        String(selectedPlantId),
    ) ?? null;

  const selectedRangeLabel =
    HISTORY_RANGES.find(
      (range) =>
        range.value ===
        selectedRange,
    )?.label ?? selectedRange;

  return (
    <main id="monitoring-report">
      <header className="no-print flex flex-col gap-5 border-b border-slate-200 pb-7 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-semibold uppercase tracking-widest text-emerald-700">
            Monitoring Tanaman
          </p>

          <h1 className="mt-2 text-3xl font-semibold tracking-tight text-slate-950">
            Dashboard kondisi tanaman
          </h1>

          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600">
            Nilai sensor dan status
            monitoring diperbarui otomatis
            setiap 30 detik.
          </p>
        </div>

        <button
          type="button"
          onClick={handleReload}
          disabled={isReloading}
          className="inline-flex h-11 items-center justify-center rounded-xl border border-slate-300 bg-white px-5 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {isReloading
            ? "Memuat..."
            : "Muat ulang semua data"}
        </button>
      </header>

      <section className="no-print mt-7">
        <p className="mb-3 text-sm font-medium text-slate-600">
          Pilih tanaman
        </p>

        {isLoadingPlants ? (
          <StatusBox>
            Mengambil daftar tanaman...
          </StatusBox>
        ) : plantListError ? (
          <ErrorBox>
            {plantListError}
          </ErrorBox>
        ) : plants.length === 0 ? (
          <StatusBox>
            Belum ada tanaman yang terdaftar.
          </StatusBox>
        ) : (
          <div className="flex flex-wrap gap-3">
            {plants.map((plant) => {
              const isSelected =
                String(
                  selectedPlantId,
                ) ===
                String(plant.id);

              return (
                <button
                  key={plant.id}
                  type="button"
                  onClick={() =>
                    setSelectedPlantId(
                      plant.id,
                    )
                  }
                  className={[
                    "rounded-xl px-4 py-2.5 text-sm font-semibold transition",

                    isSelected
                      ? "bg-slate-900 text-white shadow-sm"
                      : "border border-slate-300 bg-white text-slate-700 hover:bg-slate-50",
                  ].join(" ")}
                >
                  {plant.name}
                </button>
              );
            })}
          </div>
        )}
      </section>

      {refreshWarning && (
        <section className="mt-6 rounded-2xl border border-amber-200 bg-amber-50 px-5 py-4">
          <p className="text-sm text-amber-800">
            {refreshWarning}
          </p>

          <p className="mt-1 text-xs text-amber-700">
            Data terakhir tetap ditampilkan.
          </p>
        </section>
      )}

      {isLoadingDashboard && (
        <StatusBox className="mt-8">
          Mengambil data tanaman...
        </StatusBox>
      )}

      {!isLoadingDashboard &&
        dashboardError && (
          <ErrorBox className="mt-8">
            {dashboardError}
          </ErrorBox>
        )}

      {!isLoadingDashboard &&
        !dashboardError &&
        hasNoSensorData &&
        dashboard?.plant && (
          <section className="mt-8 rounded-2xl border border-amber-200 bg-amber-50 p-6 shadow-sm">
            <p className="text-sm font-semibold uppercase tracking-wider text-amber-700">
              Belum terhubung
            </p>

            <h2 className="mt-2 text-xl font-semibold text-slate-950">
              {dashboard.plant.name} belum
              memiliki data sensor
            </h2>

            <p className="mt-2 text-sm leading-6 text-slate-600">
              Grafik dan rekomendasi akan
              muncul setelah data sensor
              tersimpan.
            </p>
          </section>
        )}

      {!isLoadingDashboard &&
        !dashboardError &&
        hasSensorData &&
        dashboard?.plant &&
        dashboard?.sensor &&
        dashboard?.condition && (
          <div className="mt-8 space-y-6">
            <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
              <p className="text-sm font-medium text-slate-500">
                Tanaman yang dipantau
              </p>

              <h2 className="mt-1 text-2xl font-semibold text-slate-950">
                {dashboard.plant.name}
              </h2>

              <p className="mt-1 text-sm italic text-slate-500">
                {dashboard.plant.type}
              </p>
            </section>

            <MonitoringStatusCard
              monitoring={
                dashboard.monitoring
              }
              lastClientRefreshAt={
                lastClientRefreshAt
              }
              isRefreshing={
                isAutoRefreshing
              }
            />

            <section className="grid gap-4 md:grid-cols-3">
              <SensorCard
                title="Suhu Udara"
                value={
                  dashboard.sensor
                    .temperature
                }
                unit="°C"
                description="Data suhu terbaru dari sensor."
              />

              <SensorCard
                title="Kelembapan Udara"
                value={
                  dashboard.sensor
                    .humidity
                }
                unit="%"
                description="Kelembapan udara terbaru."
              />

              <SensorCard
                title="Kelembapan Tanah"
                value={
                  dashboard.sensor
                    .soil_moisture
                }
                unit="%"
                description="Kelembapan media tanam terbaru."
              />
            </section>

            <ConditionCard
              condition={
                dashboard.condition
              }
            />

            <DSSAnalysisPanel
              plantId={
                selectedPlantId
              }
              range={
                selectedRange
              }
            />

            <section className="pt-4">
              <div className="flex flex-col gap-5 border-b border-slate-200 pb-5">
                <div>
                  <p className="text-sm font-semibold uppercase tracking-wider text-emerald-700">
                    Histori Sensor
                  </p>

                  <h2 className="mt-2 text-2xl font-semibold text-slate-950">
                    Perubahan kondisi dari waktu ke waktu
                  </h2>
                </div>

                <div className="no-print flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
                  <HistoryExportButtons
                    plant={
                      selectedPlant
                    }
                    rangeLabel={
                      selectedRangeLabel
                    }
                    history={history}
                    readings={
                      historyReadings
                    }
                  />

                  <div className="flex flex-wrap gap-2">
                    {HISTORY_RANGES.map(
                      (range) => (
                        <button
                          key={
                            range.value
                          }
                          type="button"
                          onClick={() =>
                            setSelectedRange(
                              range.value,
                            )
                          }
                          className={[
                            "rounded-lg px-3 py-2 text-xs font-semibold transition",

                            selectedRange ===
                            range.value
                              ? "bg-slate-900 text-white"
                              : "border border-slate-300 bg-white text-slate-600 hover:bg-slate-50",
                          ].join(" ")}
                        >
                          {range.label}
                        </button>
                      ),
                    )}
                  </div>
                </div>
              </div>

              {isLoadingHistory ? (
                <StatusBox className="mt-6">
                  Mengambil histori sensor...
                </StatusBox>
              ) : historyError ? (
                <ErrorBox className="mt-6">
                  {historyError}
                </ErrorBox>
              ) : historyReadings.length ===
                0 ? (
                <StatusBox className="mt-6">
                  Belum ada data pada rentang waktu ini.
                </StatusBox>
              ) : (
                <div className="mt-6 space-y-6">
                  <CombinedSensorHistoryChart
                    readings={
                      historyReadings
                    }
                    rangeLabel={
                      selectedRangeLabel
                    }
                  />

                  <SensorHistoryChart
                    title="Suhu Udara"
                    description="Perubahan suhu udara selama periode terpilih."
                    readings={
                      historyReadings
                    }
                    rangeLabel={
                      selectedRangeLabel
                    }
                    dataKey="temperature"
                    unit="°C"
                    stroke="#059669"
                  />

                  <SensorHistoryChart
                    title="Kelembapan Udara"
                    description="Perubahan kelembapan udara selama periode terpilih."
                    readings={
                      historyReadings
                    }
                    rangeLabel={
                      selectedRangeLabel
                    }
                    dataKey="humidity"
                    unit="%"
                    stroke="#2563eb"
                  />

                  <SensorHistoryChart
                    title="Kelembapan Tanah"
                    description="Perubahan kelembapan media tanam selama periode terpilih."
                    readings={
                      historyReadings
                    }
                    rangeLabel={
                      selectedRangeLabel
                    }
                    dataKey="soil_moisture"
                    unit="%"
                    stroke="#d97706"
                  />
                </div>
              )}
            </section>
          </div>
        )}
    </main>
  );
}

function StatusBox({
  children,
  className = "",
}) {
  return (
    <section
      className={[
        "rounded-2xl border border-slate-200 bg-white p-6 text-center shadow-sm",
        className,
      ].join(" ")}
    >
      <p className="text-sm text-slate-500">
        {children}
      </p>
    </section>
  );
}

function ErrorBox({
  children,
  className = "",
}) {
  return (
    <section
      className={[
        "rounded-2xl border border-red-200 bg-red-50 p-6",
        className,
      ].join(" ")}
    >
      <p className="text-sm text-red-700">
        {children}
      </p>
    </section>
  );
}

export default Dashboard;