export const batchItems = <T>(items: T[], max = 10): T[][] => {
  const out: T[][] = [];
  for (let i = 0; i < items.length; i += max) out.push(items.slice(i, i + max));
  return out;
};

// Simple queue for batching events
let eventQueue: any[] = [];
let batchTimer: any = null;

export const queueEvent = (event: any, config: { batchSize: number; timeout: number }, sendFn: (batch: any[]) => Promise<void>) => {
  eventQueue.push(event);
  
  // Send immediately if batch is full
  if (eventQueue.length >= config.batchSize) {
    flushQueue(sendFn);
    return;
  }
  
  // Set timer for timeout-based sending
  if (!batchTimer) {
    batchTimer = setTimeout(() => flushQueue(sendFn), config.timeout);
  }
};

const flushQueue = async (sendFn: (batch: any[]) => Promise<void>) => {
  if (eventQueue.length === 0) return;
  
  const batch = [...eventQueue];
  eventQueue = [];
  
  if (batchTimer) {
    clearTimeout(batchTimer);
    batchTimer = null;
  }
  
  try {
    await sendFn(batch);
  } catch {
    // Silently handle batch send failures
  }
};
