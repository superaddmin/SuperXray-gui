export function formatBytes(value: number | undefined, fractionDigits = 1): string {
  if (!Number.isFinite(value) || value === undefined) {
    return '-';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  let size = Math.max(value, 0);
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  return `${size.toFixed(size >= 10 || unitIndex === 0 ? 0 : fractionDigits)} ${units[unitIndex]}`;
}

export function formatDuration(totalSeconds: number | undefined): string {
  if (!Number.isFinite(totalSeconds) || totalSeconds === undefined || totalSeconds <= 0) {
    return '-';
  }

  const days = Math.floor(totalSeconds / 86_400);
  const hours = Math.floor((totalSeconds % 86_400) / 3_600);
  const minutes = Math.floor((totalSeconds % 3_600) / 60);

  if (days > 0) {
    return `${days}d ${hours}h`;
  }
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  return `${minutes}m`;
}

export function formatPercent(value: number | undefined): string {
  if (!Number.isFinite(value) || value === undefined) {
    return '-';
  }
  return `${Math.max(0, Math.min(100, value)).toFixed(1)}%`;
}

export function formatCount(value: number | undefined): string {
  if (!Number.isFinite(value) || value === undefined) {
    return '-';
  }
  return new Intl.NumberFormat().format(value);
}

export function formatDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}
