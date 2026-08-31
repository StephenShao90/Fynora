export type Guide = {
  number: number;
  title: string;
  body: string;
};

export function GuideMarker(_: { guide: Guide }) {
  return null;
}
