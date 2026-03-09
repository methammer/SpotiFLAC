export function EventsOn(_event: string, _callback: (...data: any[]) => void): () => void { return () => {}; }
export function EventsOff(..._event: string[]): void {}
export function EventsEmit(_event: string, ..._data: any[]): void {}
export function WindowSetTitle(_title: string): void {}
export function WindowMinimise(): void {}
export function WindowToggleMaximise(): void {}
export function BrowserOpenURL(_url: string): void {}
export function Quit(): void {}
