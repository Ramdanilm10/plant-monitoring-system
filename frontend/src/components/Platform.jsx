import {
  useEffect,
  useState,
} from "react";

const emptyForm = {
  name: "",
  type: "",
  min_temperature: "",
  max_temperature: "",
  min_humidity: "",
  max_humidity: "",
  min_soil_moisture: "",
  max_soil_moisture: "",
};

function createFormState(initialData) {
  if (!initialData) {
    return emptyForm;
  }

  return {
    name: initialData.name ?? "",
    type: initialData.type ?? "",
    min_temperature:
      initialData.min_temperature ?? "",
    max_temperature:
      initialData.max_temperature ?? "",
    min_humidity:
      initialData.min_humidity ?? "",
    max_humidity:
      initialData.max_humidity ?? "",
    min_soil_moisture:
      initialData.min_soil_moisture ?? "",
    max_soil_moisture:
      initialData.max_soil_moisture ?? "",
  };
}

function NumberInput({
  label,
  name,
  value,
  onChange,
  unit,
}) {
  return (
    <div>
      <label
        htmlFor={name}
        className="text-sm font-semibold text-slate-700"
      >
        {label}
      </label>

      <div className="relative mt-2">
        <input
          id={name}
          name={name}
          type="number"
          step="0.1"
          required
          value={value}
          onChange={onChange}
          className="h-11 w-full rounded-xl border border-slate-300 bg-white px-4 pr-12 text-sm text-slate-900 outline-none transition focus:border-emerald-600 focus:ring-4 focus:ring-emerald-100"
        />

        <span className="pointer-events-none absolute inset-y-0 right-4 flex items-center text-xs font-medium text-slate-400">
          {unit}
        </span>
      </div>
    </div>
  );
}

function PlantForm({
  initialData,
  isSubmitting,
  onSubmit,
  onCancel,
}) {
  const [formData, setFormData] =
    useState(emptyForm);

  useEffect(() => {
    setFormData(
      createFormState(initialData),
    );
  }, [initialData]);

  function handleChange(event) {
    const { name, value } = event.target;

    setFormData((currentData) => ({
      ...currentData,
      [name]: value,
    }));
  }

  function handleSubmit(event) {
    event.preventDefault();

    onSubmit({
      name: formData.name.trim(),
      type: formData.type.trim(),

      min_temperature: Number(
        formData.min_temperature,
      ),

      max_temperature: Number(
        formData.max_temperature,
      ),

      min_humidity: Number(
        formData.min_humidity,
      ),

      max_humidity: Number(
        formData.max_humidity,
      ),

      min_soil_moisture: Number(
        formData.min_soil_moisture,
      ),

      max_soil_moisture: Number(
        formData.max_soil_moisture,
      ),
    });
  }

  const isEditing = Boolean(initialData);

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm"
    >
      <div>
        <p className="text-sm font-semibold uppercase tracking-wider text-emerald-700">
          {isEditing
            ? "Edit Tanaman"
            : "Tambah Tanaman"}
        </p>

        <h2 className="mt-2 text-xl font-semibold text-slate-950">
          {isEditing
            ? initialData.name
            : "Data tanaman baru"}
        </h2>
      </div>

      <div className="mt-6 grid gap-5 md:grid-cols-2">
        <div>
          <label
            htmlFor="name"
            className="text-sm font-semibold text-slate-700"
          >
            Nama tanaman
          </label>

          <input
            id="name"
            name="name"
            type="text"
            required
            value={formData.name}
            onChange={handleChange}
            placeholder="Contoh: Lidah Mertua"
            className="mt-2 h-11 w-full rounded-xl border border-slate-300 bg-white px-4 text-sm outline-none transition focus:border-emerald-600 focus:ring-4 focus:ring-emerald-100"
          />
        </div>

        <div>
          <label
            htmlFor="type"
            className="text-sm font-semibold text-slate-700"
          >
            Jenis tanaman
          </label>

          <input
            id="type"
            name="type"
            type="text"
            required
            value={formData.type}
            onChange={handleChange}
            placeholder="Contoh: Sansevieria"
            className="mt-2 h-11 w-full rounded-xl border border-slate-300 bg-white px-4 text-sm outline-none transition focus:border-emerald-600 focus:ring-4 focus:ring-emerald-100"
          />
        </div>
      </div>

      <div className="mt-7">
        <h3 className="font-semibold text-slate-900">
          Batas suhu
        </h3>

        <div className="mt-4 grid gap-5 md:grid-cols-2">
          <NumberInput
            label="Minimum suhu"
            name="min_temperature"
            value={formData.min_temperature}
            onChange={handleChange}
            unit="°C"
          />

          <NumberInput
            label="Maksimum suhu"
            name="max_temperature"
            value={formData.max_temperature}
            onChange={handleChange}
            unit="°C"
          />
        </div>
      </div>

      <div className="mt-7">
        <h3 className="font-semibold text-slate-900">
          Batas kelembapan udara
        </h3>

        <div className="mt-4 grid gap-5 md:grid-cols-2">
          <NumberInput
            label="Minimum kelembapan"
            name="min_humidity"
            value={formData.min_humidity}
            onChange={handleChange}
            unit="%"
          />

          <NumberInput
            label="Maksimum kelembapan"
            name="max_humidity"
            value={formData.max_humidity}
            onChange={handleChange}
            unit="%"
          />
        </div>
      </div>

      <div className="mt-7">
        <h3 className="font-semibold text-slate-900">
          Batas kelembapan tanah
        </h3>

        <div className="mt-4 grid gap-5 md:grid-cols-2">
          <NumberInput
            label="Minimum kelembapan tanah"
            name="min_soil_moisture"
            value={
              formData.min_soil_moisture
            }
            onChange={handleChange}
            unit="%"
          />

          <NumberInput
            label="Maksimum kelembapan tanah"
            name="max_soil_moisture"
            value={
              formData.max_soil_moisture
            }
            onChange={handleChange}
            unit="%"
          />
        </div>
      </div>

      <div className="mt-8 flex flex-wrap justify-end gap-3">
        <button
          type="button"
          onClick={onCancel}
          disabled={isSubmitting}
          className="rounded-xl border border-slate-300 bg-white px-5 py-2.5 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:opacity-50"
        >
          Batal
        </button>

        <button
          type="submit"
          disabled={isSubmitting}
          className="rounded-xl bg-slate-900 px-5 py-2.5 text-sm font-semibold text-white hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {isSubmitting
            ? "Menyimpan..."
            : isEditing
              ? "Simpan perubahan"
              : "Tambah tanaman"}
        </button>
      </div>
    </form>
  );
}

export default PlantForm;