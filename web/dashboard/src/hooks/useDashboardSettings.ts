import { useEffect, useState } from "react";
import {
  defaultDashboardSettings,
  settingsApi,
  type DashboardSettings,
} from "../api/settings";

export function useDashboardSettings() {
  const [settings, setSettings] = useState<DashboardSettings>(defaultDashboardSettings);

  useEffect(() => {
    let mounted = true;
    settingsApi
      .get()
      .then((value) => {
        if (mounted) setSettings(value);
      })
      .catch(() => {
        if (mounted) setSettings(defaultDashboardSettings);
      });
    return () => {
      mounted = false;
    };
  }, []);

  return settings;
}
