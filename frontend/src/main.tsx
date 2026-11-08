import React from "react";
import ReactDOM from "react-dom/client";
import "@fontsource/inter/400.css";
import "@fontsource/inter/600.css";
import "@fontsource/inter/700.css";
import "@fontsource/space-grotesk/600.css";
import "@fontsource/space-grotesk/700.css";
import "./styles.css";
import App from "./App";
import { DataProvider } from "./data";
import { FinishedCardProvider } from "./components/FinishedCard";
import { ToastProvider } from "./toast";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <DataProvider>
      <ToastProvider>
        <FinishedCardProvider>
          <App />
        </FinishedCardProvider>
      </ToastProvider>
    </DataProvider>
  </React.StrictMode>,
);
