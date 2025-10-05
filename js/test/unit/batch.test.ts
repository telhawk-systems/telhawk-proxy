import { batchItems, queueEvent } from '../../src/transport/batch';

describe('Batch utilities', () => {
  describe('batchItems', () => {
    test('batches items into groups of 10 by default', () => {
      const items = Array.from({ length: 25 }, (_, i) => i);
      const batches = batchItems(items);
      
      expect(batches).toHaveLength(3);
      expect(batches[0]).toHaveLength(10);
      expect(batches[1]).toHaveLength(10);
      expect(batches[2]).toHaveLength(5);
    });

    test('batches items with custom max size', () => {
      const items = Array.from({ length: 17 }, (_, i) => i);
      const batches = batchItems(items, 5);
      
      expect(batches).toHaveLength(4);
      expect(batches[0]).toHaveLength(5);
      expect(batches[1]).toHaveLength(5);
      expect(batches[2]).toHaveLength(5);
      expect(batches[3]).toHaveLength(2);
    });

    test('handles empty array', () => {
      const batches = batchItems([]);
      expect(batches).toHaveLength(0);
      expect(batches).toEqual([]);
    });

    test('handles single item', () => {
      const batches = batchItems(['one']);
      expect(batches).toHaveLength(1);
      expect(batches[0]).toEqual(['one']);
    });

    test('handles exact multiple of batch size', () => {
      const items = Array.from({ length: 20 }, (_, i) => i);
      const batches = batchItems(items, 10);
      
      expect(batches).toHaveLength(2);
      expect(batches[0]).toHaveLength(10);
      expect(batches[1]).toHaveLength(10);
    });
  });

  describe('queueEvent', () => {
    beforeEach(() => {
      jest.clearAllTimers();
      jest.useFakeTimers();
    });

    afterEach(() => {
      jest.useRealTimers();
    });

    test('queues events and sends when batch is full', async () => {
      const sendFn = jest.fn().mockResolvedValue(undefined);
      const config = { batchSize: 3, timeout: 1000 };

      queueEvent({ id: 1 }, config, sendFn);
      queueEvent({ id: 2 }, config, sendFn);
      expect(sendFn).not.toHaveBeenCalled();

      queueEvent({ id: 3 }, config, sendFn);
      
      // Give promises time to resolve
      await Promise.resolve();
      
      expect(sendFn).toHaveBeenCalledTimes(1);
      expect(sendFn).toHaveBeenCalledWith([
        { id: 1 },
        { id: 2 },
        { id: 3 }
      ]);
    });

    test('sends events on timeout', async () => {
      const sendFn = jest.fn().mockResolvedValue(undefined);
      const config = { batchSize: 5, timeout: 1000 };

      queueEvent({ id: 1 }, config, sendFn);
      queueEvent({ id: 2 }, config, sendFn);
      
      expect(sendFn).not.toHaveBeenCalled();
      
      jest.advanceTimersByTime(1000);
      await Promise.resolve();
      
      expect(sendFn).toHaveBeenCalledTimes(1);
      expect(sendFn).toHaveBeenCalledWith([
        { id: 1 },
        { id: 2 }
      ]);
    });

    test('handles send failures silently', async () => {
      const sendFn = jest.fn().mockRejectedValue(new Error('Network error'));
      const config = { batchSize: 2, timeout: 1000 };

      queueEvent({ id: 1 }, config, sendFn);
      queueEvent({ id: 2 }, config, sendFn);
      
      await Promise.resolve();
      
      expect(sendFn).toHaveBeenCalledTimes(1);
      // Should not throw
    });
  });
});
