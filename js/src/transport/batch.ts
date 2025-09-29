export const batchItems = <T>(items: T[], max = 10): T[][] => {
  const out: T[][] = [];
  for (let i = 0; i < items.length; i += max) out.push(items.slice(i, i + max));
  return out;
};
