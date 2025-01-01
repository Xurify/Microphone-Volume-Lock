import React from "react";
import { createRoot } from "react-dom/client";
import "./style.scss";
import App from "./App";
import { Theme } from "@radix-ui/themes";
import "@radix-ui/themes/styles.css";

const container = document.getElementById("root") as HTMLDivElement;
const root = createRoot(container);

root.render(
  <React.StrictMode>
    <Theme accentColor="blue" grayColor="sand" radius="large" scaling="95%" appearance="dark">
      <App />
    </Theme>
  </React.StrictMode>
);
