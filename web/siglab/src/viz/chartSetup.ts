// One-time Chart.js registration shared by every Chart.js-backed panel.
import {
  BarController,
  BarElement,
  CategoryScale,
  Chart,
  Filler,
  Legend,
  LineController,
  LineElement,
  LinearScale,
  PointElement,
  ScatterController,
  Title,
  Tooltip,
} from "chart.js";

let registered = false;

export function ensureChart(): void {
  if (registered) return;
  Chart.register(
    BarController,
    BarElement,
    LineController,
    LineElement,
    ScatterController,
    PointElement,
    CategoryScale,
    LinearScale,
    Filler,
    Legend,
    Title,
    Tooltip,
  );
  registered = true;
}
