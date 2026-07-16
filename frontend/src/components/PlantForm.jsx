import {
  useEffect,
  useState,
} from "react";

const EMPTY_FORM = {
  name: "",
  type: "",
  min_temperature: "",
  max_temperature: "",
  min_humidity: "",
  max_humidity: "",
  min_soil_moisture: "",
  max_soil_moisture: "",
};

const NUMBER_FIELDS = [
  "min_temperature",
  "max_temperature",
  "min_humidity",
  "max_humidity",
  "min_soil_moisture",
  "max_soil_moisture",
];

function createFormData(initialData) {
  if (!initialData) {
    return EMPTY_FORM;
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

function PlantForm({
  initialData = null,
  isSubmitting = false,
  onSubmit,
  onCancel,
}) {
  const [formData, setFormData] =
    useState(EMPTY_FORM);

  const [validationMessage, setValidationMessage] =
    useState("");

  const isEditing = Boolean(initialData);

  useEffect(() => {
    setFormData(createFormData(initialData));
    setValidationMessage("");
  }, [initialData]);

  function handleChange(event) {
    const { name, value } = event.target;

    setFormData((currentData) => ({
      ...currentData,
      [name]: value,
    }));

    setValidationMessage("");
  }

  function validateForm() {
    if (!formData.name.trim()) {
      return "Nama tanaman wajib diisi.";
    }

    if (!formData.type.trim()) {
      return "Jenis tanaman wajib diisi.";
    }

    for (const fieldName of NUMBER_FIELDS) {
      const value = formData[fieldName];

      if (
        value === "" ||
        value === null ||
        value === undefined
      ) {
        return "Semua nilai batas kondisi wajib diisi.";
      }

      const numericValue = Number(value);

      if (!Number.isFinite(numericValue)) {
        return "Batas kondisi harus berupa angka.";
      }
    }

    const minTemperature = Number(
      formData.min_temperature,
    );

    const maxTemperature = Number(
      formData.max_temperature,
    );

    const minHumidity = Number(
      formData.min_humidity,
    );

    const maxHumidity = Number(
      formData.max_humidity,
    );

    const minSoilMoisture = Number(
      formData.min_soil_moisture,
    );

    const maxSoilMoisture = Number(
      formData.max_soil_moisture,
    );

    if (minTemperature > maxTemperature) {
      return "Suhu minimum tidak boleh lebih besar dari suhu maksimum.";
    }

    if (minHumidity > maxHumidity) {
      return "Kelembapan udara minimum tidak boleh lebih besar dari nilai maksimum.";
    }

    if (
      minSoilMoisture >
      maxSoilMoisture
    ) {
      return "Kelembapan tanah minimum tidak boleh lebih besar dari nilai maksimum.";
    }

    if (
      minHumidity < 0 ||
      maxHumidity > 100
    ) {
      return "Kelembapan udara harus berada di antara 0 sampai 100 persen.";
    }

    if (
      minSoilMoisture < 0 ||
      maxSoilMoisture > 100
    ) {
      return "Kelembapan tanah harus berada di antara 0 sampai 100 persen.";
    }

    return "";
  }

  async function handleSubmit(event) {
    event.preventDefault();

    const error = validateForm();

    if (error) {
      setValidationMessage(error);
      return;
    }

    const payload = {
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
    };

    await onSubmit(payload);
  }

  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <div>
        <p className="text-sm font-semibold uppercase tracking-wider text-emerald-700">
          Form tanaman
        </p>

        <h2 className="mt-2 text-xl font-semibold text-slate-950">
          {isEditing
            ? "Edit tanaman"
            : "Tambah tanaman"}
        </h2>

        <p className="mt-2 text-sm leading-6 text-slate-600">
          Masukkan identitas tanaman dan batas
          kondisi idealnya.
        </p>
      </div>

      {validationMessage && (
        <div className="mt-5 rounded-xl border border-red-200 bg-red-50 px-4 py-3">
          <p className="text-sm text-red-700">
            {validationMessage}
          </p>
        </div>
      )}

      <form
        onSubmit={handleSubmit}
        className="mt-6 space-y-6"
      >
        <div className="grid gap-5 md:grid-cols-2">
          <FormInput
            label="Nama tanaman"
            name="name"
            value={formData.name}
            placeholder="Contoh: Cabai Merah"
            disabled={isSubmitting}
            onChange={handleChange}
          />

          <FormInput
            label="Jenis tanaman"
            name="type"
            value={formData.type}
            placeholder="Contoh: Hortikultura"
            disabled={isSubmitting}
            onChange={handleChange}
          />
        </div>

        <ConditionGroup
          title="Suhu udara"
          unit="°C"
          minName="min_temperature"
          maxName="max_temperature"
          minValue={formData.min_temperature}
          maxValue={formData.max_temperature}
          disabled={isSubmitting}
          onChange={handleChange}
        />

        <ConditionGroup
          title="Kelembapan udara"
          unit="%"
          minName="min_humidity"
          maxName="max_humidity"
          minValue={formData.min_humidity}
          maxValue={formData.max_humidity}
          disabled={isSubmitting}
          onChange={handleChange}
          min={0}
          max={100}
        />

        <ConditionGroup
          title="Kelembapan tanah"
          unit="%"
          minName="min_soil_moisture"
          maxName="max_soil_moisture"
          minValue={
            formData.min_soil_moisture
          }
          maxValue={
            formData.max_soil_moisture
          }
          disabled={isSubmitting}
          onChange={handleChange}
          min={0}
          max={100}
        />

        <div className="flex flex-col-reverse gap-3 border-t border-slate-200 pt-5 sm:flex-row sm:justify-end">
          <button
            type="button"
            onClick={onCancel}
            disabled={isSubmitting}
            className="rounded-xl border border-slate-300 px-5 py-3 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            Batal
          </button>

          <button
            type="submit"
            disabled={isSubmitting}
            className="rounded-xl bg-emerald-700 px-5 py-3 text-sm font-semibold text-white transition hover:bg-emerald-800 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isSubmitting
              ? "Menyimpan..."
              : isEditing
                ? "Simpan perubahan"
                : "Tambah tanaman"}
          </button>
        </div>
      </form>
    </section>
  );
}

function FormInput({
  label,
  name,
  value,
  placeholder,
  disabled,
  onChange,
}) {
  return (
    <label className="block">
      <span className="mb-2 block text-sm font-semibold text-slate-700">
        {label}
      </span>

      <input
        type="text"
        name={name}
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        onChange={onChange}
        autoComplete="off"
        className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-emerald-600 focus:ring-4 focus:ring-emerald-100 disabled:cursor-not-allowed disabled:bg-slate-100"
      />
    </label>
  );
}

function ConditionGroup({
  title,
  unit,
  minName,
  maxName,
  minValue,
  maxValue,
  disabled,
  onChange,
  min,
  max,
}) {
  return (
    <fieldset className="rounded-xl border border-slate-200 p-5">
      <legend className="px-2 text-sm font-semibold text-slate-800">
        {title}
      </legend>

      <div className="grid gap-5 sm:grid-cols-2">
        <NumberInput
          label={`Minimum (${unit})`}
          name={minName}
          value={minValue}
          disabled={disabled}
          onChange={onChange}
          min={min}
          max={max}
        />

        <NumberInput
          label={`Maksimum (${unit})`}
          name={maxName}
          value={maxValue}
          disabled={disabled}
          onChange={onChange}
          min={min}
          max={max}
        />
      </div>
    </fieldset>
  );
}

function NumberInput({
  label,
  name,
  value,
  disabled,
  onChange,
  min,
  max,
}) {
  return (
    <label className="block">
      <span className="mb-2 block text-sm font-medium text-slate-700">
        {label}
      </span>

      <input
        type="number"
        name={name}
        value={value}
        disabled={disabled}
        onChange={onChange}
        min={min}
        max={max}
        step="0.1"
        inputMode="decimal"
        className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none transition focus:border-emerald-600 focus:ring-4 focus:ring-emerald-100 disabled:cursor-not-allowed disabled:bg-slate-100"
      />
    </label>
  );
}

export default PlantForm;