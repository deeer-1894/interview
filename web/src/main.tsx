import React from "react";
import ReactDOM from "react-dom/client";

import App from "@/app";
import "@/styles/globals.css";

class RootErrorBoundary extends React.Component<
  { children: React.ReactNode },
  { hasError: boolean; errorMessage: string }
> {
  constructor(props: { children: React.ReactNode }) {
    super(props);
    this.state = { hasError: false, errorMessage: "" };
  }

  static getDerivedStateFromError(error: unknown) {
    return {
      hasError: true,
      errorMessage: error instanceof Error ? error.message : "前端渲染发生未知错误",
    };
  }

  componentDidCatch(error: unknown) {
    console.error("root render failed", error);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div
          style={{
            minHeight: "100vh",
            display: "grid",
            placeItems: "center",
            background: "linear-gradient(180deg, #f4f7fc 0%, #eef3fb 42%, #eaf1f8 100%)",
            padding: "24px",
            color: "#485b72",
            fontFamily: '"Noto Sans SC", sans-serif',
          }}
        >
          <div
            style={{
              width: "min(560px, 100%)",
              borderRadius: "24px",
              border: "1px solid rgba(176,194,221,0.6)",
              background: "rgba(255,255,255,0.94)",
              boxShadow: "0 18px 42px rgba(107,132,167,0.14)",
              padding: "24px",
            }}
          >
            <p style={{ margin: 0, fontSize: "12px", letterSpacing: "0.16em", textTransform: "uppercase", color: "rgba(97,123,150,0.74)" }}>
              Frontend Error
            </p>
            <h1 style={{ margin: "12px 0 0", fontSize: "28px", lineHeight: 1.2, fontFamily: '"Sora", "Noto Sans SC", sans-serif' }}>
              页面渲染失败
            </h1>
            <p style={{ margin: "16px 0 0", fontSize: "14px", lineHeight: 1.8, color: "rgba(115,137,161,0.9)" }}>
              {this.state.errorMessage || "请查看浏览器控制台里的报错信息。"}
            </p>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <RootErrorBoundary>
    <App />
  </RootErrorBoundary>,
);
