import { Component } from "react";

class AppErrorBoundary extends Component {
  constructor(props) {
    super(props);

    this.state = {
      hasError: false,
      error: null,
    };
  }

  static getDerivedStateFromError(error) {
    return {
      hasError: true,
      error,
    };
  }

  componentDidCatch(error, errorInfo) {
    console.error(
      "Aplikasi gagal dirender:",
      error,
      errorInfo,
    );
  }

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (!this.state.hasError) {
      return this.props.children;
    }

    return (
      <main className="min-h-screen bg-slate-100 p-6">
        <section className="mx-auto max-w-3xl rounded-2xl border border-red-200 bg-white p-6 shadow-sm">
          <p className="text-sm font-semibold uppercase tracking-wider text-red-700">
            React gagal merender aplikasi
          </p>

          <h1 className="mt-2 text-2xl font-semibold text-slate-950">
            Terjadi kesalahan pada frontend
          </h1>

          <p className="mt-3 text-sm leading-6 text-slate-600">
            Error aplikasi ditampilkan di bawah
            agar halaman tidak hanya menjadi
            layar putih tanpa penjelasan.
          </p>

          <pre className="mt-5 overflow-x-auto whitespace-pre-wrap rounded-xl bg-slate-950 p-4 text-sm leading-6 text-red-200">
            {this.state.error?.stack ||
              this.state.error?.message ||
              String(this.state.error)}
          </pre>

          <button
            type="button"
            onClick={this.handleReload}
            className="mt-5 rounded-xl bg-slate-900 px-5 py-3 text-sm font-semibold text-white transition hover:bg-slate-800"
          >
            Muat ulang aplikasi
          </button>
        </section>
      </main>
    );
  }
}

export default AppErrorBoundary;