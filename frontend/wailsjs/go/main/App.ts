export function CheckFFmpegInstalled(): Promise<boolean> { return Promise.resolve(true); }
export function DownloadFFmpeg(): Promise<void> { return Promise.resolve(); }
export function OpenFolder(_path: string): Promise<void> { return Promise.resolve(); }
export function GetPreviewURL(_id: string): Promise<string> { return Promise.resolve(""); }
