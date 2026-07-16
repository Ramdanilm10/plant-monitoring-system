import {
  useCallback,
  useEffect,
  useState,
} from "react";

import PlantForm from "../components/PlantForm";

import {
  createPlant,
  deletePlant,
  getPlants,
  updatePlant,
} from "../services/api";

function getPlantID(plant) {
  return plant?.id ?? plant?._id ?? null;
}

function AdminPlants() {
  const [plants, setPlants] = useState([]);
  const [editingPlant, setEditingPlant] =
    useState(null);

  const [isFormOpen, setIsFormOpen] =
    useState(false);

  const [isLoading, setIsLoading] =
    useState(true);

  const [isSubmitting, setIsSubmitting] =
    useState(false);

  const [deletingID, setDeletingID] =
    useState(null);

  const [errorMessage, setErrorMessage] =
    useState("");

  const [successMessage, setSuccessMessage] =
    useState("");

  const loadPlants = useCallback(async () => {
    setIsLoading(true);
    setErrorMessage("");

    try {
      const result = await getPlants();

      const plantList = Array.isArray(result)
        ? result
        : result?.data;

      setPlants(
        Array.isArray(plantList)
          ? plantList
          : [],
      );
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "Daftar tanaman gagal dimuat.",
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadPlants();
  }, [loadPlants]);

  function openCreateForm() {
    setEditingPlant(null);
    setIsFormOpen(true);
    setErrorMessage("");
    setSuccessMessage("");
  }

  function openEditForm(plant) {
    setEditingPlant(plant);
    setIsFormOpen(true);
    setErrorMessage("");
    setSuccessMessage("");

    window.scrollTo({
      top: 0,
      behavior: "smooth",
    });
  }

  function closeForm() {
    setEditingPlant(null);
    setIsFormOpen(false);
  }

  async function handleSubmit(plantData) {
    setIsSubmitting(true);
    setErrorMessage("");
    setSuccessMessage("");

    try {
      let result;

      if (editingPlant) {
        const plantID =
          getPlantID(editingPlant);

        if (!plantID) {
          throw new Error(
            "ID tanaman tidak ditemukan.",
          );
        }

        result = await updatePlant(
          plantID,
          plantData,
        );

        setSuccessMessage(
          result?.message ||
            "Data tanaman berhasil diperbarui.",
        );
      } else {
        result = await createPlant(plantData);

        setSuccessMessage(
          result?.message ||
            "Tanaman berhasil ditambahkan.",
        );
      }

      closeForm();
      await loadPlants();
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "Data tanaman gagal disimpan.",
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleDelete(plant) {
    const plantID = getPlantID(plant);

    if (!plantID) {
      setErrorMessage(
        "ID tanaman tidak ditemukan.",
      );
      return;
    }

    const confirmed = window.confirm(
      `Hapus tanaman "${plant.name}"?`,
    );

    if (!confirmed) {
      return;
    }

    setDeletingID(plantID);
    setErrorMessage("");
    setSuccessMessage("");

    try {
      const result = await deletePlant(
        plantID,
      );

      setSuccessMessage(
        result?.message ||
          "Tanaman berhasil dihapus.",
      );

      if (
        getPlantID(editingPlant) === plantID
      ) {
        closeForm();
      }

      await loadPlants();
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "Tanaman gagal dihapus.",
      );
    } finally {
      setDeletingID(null);
    }
  }

  return (
    <main>
      <header className="flex flex-col gap-4 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-sm font-semibold uppercase tracking-wider text-emerald-700">
            Admin
          </p>

          <h1 className="mt-2 text-2xl font-semibold text-slate-950">
            Kelola tanaman
          </h1>

          <p className="mt-2 text-sm leading-6 text-slate-600">
            Tambah, ubah, dan hapus tanaman
            beserta batas kondisi idealnya.
          </p>
        </div>

        <button
          type="button"
          onClick={openCreateForm}
          disabled={isSubmitting}
          className="rounded-xl bg-slate-900 px-5 py-3 text-sm font-semibold text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-50"
        >
          Tambah tanaman
        </button>
      </header>

      {errorMessage && (
        <div className="mt-5 rounded-xl border border-red-200 bg-red-50 px-5 py-4">
          <p className="text-sm text-red-700">
            {errorMessage}
          </p>
        </div>
      )}

      {successMessage && (
        <div className="mt-5 rounded-xl border border-emerald-200 bg-emerald-50 px-5 py-4">
          <p className="text-sm text-emerald-700">
            {successMessage}
          </p>
        </div>
      )}

      {isFormOpen && (
        <div className="mt-6">
          <PlantForm
            initialData={editingPlant}
            isSubmitting={isSubmitting}
            onSubmit={handleSubmit}
            onCancel={closeForm}
          />
        </div>
      )}

      <section className="mt-6 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        {isLoading ? (
          <div className="p-8 text-center text-sm text-slate-500">
            Mengambil daftar tanaman...
          </div>
        ) : plants.length === 0 ? (
          <div className="p-8 text-center text-sm text-slate-500">
            Belum ada tanaman.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200">
              <thead className="bg-slate-50">
                <tr>
                  <th className="px-5 py-4 text-left text-xs font-semibold uppercase tracking-wider text-slate-500">
                    Tanaman
                  </th>

                  <th className="px-5 py-4 text-left text-xs font-semibold uppercase tracking-wider text-slate-500">
                    Suhu
                  </th>

                  <th className="px-5 py-4 text-left text-xs font-semibold uppercase tracking-wider text-slate-500">
                    Udara
                  </th>

                  <th className="px-5 py-4 text-left text-xs font-semibold uppercase tracking-wider text-slate-500">
                    Tanah
                  </th>

                  <th className="px-5 py-4 text-right text-xs font-semibold uppercase tracking-wider text-slate-500">
                    Aksi
                  </th>
                </tr>
              </thead>

              <tbody className="divide-y divide-slate-100">
                {plants.map((plant) => {
                  const plantID =
                    getPlantID(plant);

                  return (
                    <tr
                      key={
                        plantID ??
                        `${plant.name}-${plant.type}`
                      }
                    >
                      <td className="px-5 py-4">
                        <p className="font-semibold text-slate-900">
                          {plant.name}
                        </p>

                        <p className="mt-1 text-sm text-slate-500">
                          {plant.type}
                        </p>
                      </td>

                      <td className="whitespace-nowrap px-5 py-4 text-sm text-slate-600">
                        {plant.min_temperature}
                        {" – "}
                        {plant.max_temperature}
                        °C
                      </td>

                      <td className="whitespace-nowrap px-5 py-4 text-sm text-slate-600">
                        {plant.min_humidity}
                        {" – "}
                        {plant.max_humidity}%
                      </td>

                      <td className="whitespace-nowrap px-5 py-4 text-sm text-slate-600">
                        {
                          plant.min_soil_moisture
                        }
                        {" – "}
                        {
                          plant.max_soil_moisture
                        }
                        %
                      </td>

                      <td className="whitespace-nowrap px-5 py-4 text-right">
                        <button
                          type="button"
                          onClick={() =>
                            openEditForm(plant)
                          }
                          disabled={
                            isSubmitting ||
                            deletingID === plantID
                          }
                          className="mr-2 rounded-lg border border-slate-300 px-3 py-2 text-xs font-semibold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          Edit
                        </button>

                        <button
                          type="button"
                          disabled={
                            deletingID === plantID ||
                            isSubmitting
                          }
                          onClick={() =>
                            handleDelete(plant)
                          }
                          className="rounded-lg border border-red-200 px-3 py-2 text-xs font-semibold text-red-700 transition hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          {deletingID === plantID
                            ? "Menghapus..."
                            : "Hapus"}
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </main>
  );
}

export default AdminPlants;