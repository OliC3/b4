export const formatBytes = (
  bytes: number | string | null | undefined
): string => {
  // Handle null/undefined/invalid values
  if (bytes === null || bytes === undefined) return "0 B";

  // Convert to number safely
  const num = typeof bytes === "string" ? parseFloat(bytes) : Number(bytes);

  // Check for invalid numbers
  if (isNaN(num) || !isFinite(num) || num < 0) return "0 B";

  if (num === 0) return "0 B";

  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.min(Math.floor(Math.log(num) / Math.log(k)), sizes.length - 1);
  const value = num / Math.pow(k, i);

  // Safe formatting
  return (isFinite(value) ? value.toFixed(2) : "0") + " " + sizes[i];
};

export const formatNumber = (
  num: number | string | null | undefined
): string => {
  // Handle null/undefined/invalid values
  if (num === null || num === undefined) return "0";

  // Convert to number safely
  const value = typeof num === "string" ? parseFloat(num) : Number(num);

  // Check for invalid numbers
  if (isNaN(value) || !isFinite(value)) return "0";

  // Handle negative numbers
  const absValue = Math.abs(value);
  const sign = value < 0 ? "-" : "";

  if (absValue >= 1000000) {
    const formatted = (absValue / 1000000).toFixed(1);
    return sign + (isFinite(parseFloat(formatted)) ? formatted : "0") + "M";
  }
  if (absValue >= 1000) {
    const formatted = (absValue / 1000).toFixed(1);
    return sign + (isFinite(parseFloat(formatted)) ? formatted : "0") + "K";
  }

  return sign + Math.floor(absValue).toString();
};
