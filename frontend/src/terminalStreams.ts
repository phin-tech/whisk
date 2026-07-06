export type TerminalStreamState<Stream = unknown, Snapshot = unknown> = {
  outputChunks: Record<string, Uint8Array[]>;
  outputChunkStartOffsets: Record<string, number[]>;
  terminalSnapshots: Record<string, Snapshot>;
  offsets: Record<string, number>;
  bottomJumpRevisions: Record<string, number>;
  ptyStreams: Record<string, Stream>;
  ptyReconnectTimers: Record<string, number>;
  ptyReconnectAttempts: Record<string, number>;
  outputFetchInFlight: ReadonlyMap<string, number>;
  outputFetchAgain: ReadonlySet<string>;
  ptyDialInFlight: ReadonlySet<string>;
};

export type TerminalStreamReadModel = {
  ptys?: readonly (TerminalStreamPtyReadModel | null | undefined)[] | null;
};

export type TerminalStreamPtyReadModel = {
  id?: string | null;
};

export type TerminalStreamCleanup<Stream = unknown, Snapshot = unknown> = {
  removedPtyIds: string[];
  streamsToClose: Stream[];
  reconnectTimersToClear: number[];
  nextState: TerminalStreamState<Stream, Snapshot>;
};

export function terminalStreamHasLivePty(ptyId: string, livePtyIds: Iterable<string>) {
  const normalized = normalizedPtyId(ptyId);
  return Boolean(normalized) && normalizedPtyIdSet(livePtyIds).has(normalized);
}

export function terminalStreamCleanupForLivePtys<Stream, Snapshot>(
  state: TerminalStreamState<Stream, Snapshot>,
  livePtyIds: Iterable<string>,
): TerminalStreamCleanup<Stream, Snapshot> {
  const live = normalizedPtyIdSet(livePtyIds);
  const removedPtyIds = terminalStreamRemovedPtyIds(state, live);
  return {
    removedPtyIds,
    streamsToClose: removedPtyIds.flatMap((ptyId) => {
      const stream = state.ptyStreams[ptyId];
      return stream === undefined ? [] : [stream];
    }),
    reconnectTimersToClear: removedPtyIds.flatMap((ptyId) => {
      const timer = state.ptyReconnectTimers[ptyId];
      return timer === undefined ? [] : [timer];
    }),
    nextState: retainPtyState(state, live),
  };
}

export function retainPtyState<Stream, Snapshot>(
  state: TerminalStreamState<Stream, Snapshot>,
  livePtyIds: Iterable<string>,
): TerminalStreamState<Stream, Snapshot> {
  const live = normalizedPtyIdSet(livePtyIds);
  return {
    outputChunks: retainRecord(state.outputChunks, live),
    outputChunkStartOffsets: retainRecord(state.outputChunkStartOffsets, live),
    terminalSnapshots: retainRecord(state.terminalSnapshots, live),
    offsets: retainRecord(state.offsets, live),
    bottomJumpRevisions: retainRecord(state.bottomJumpRevisions, live),
    ptyStreams: retainRecord(state.ptyStreams, live),
    ptyReconnectTimers: retainRecord(state.ptyReconnectTimers, live),
    ptyReconnectAttempts: retainRecord(state.ptyReconnectAttempts, live),
    outputFetchInFlight: retainMap(state.outputFetchInFlight, live),
    outputFetchAgain: retainSet(state.outputFetchAgain, live),
    ptyDialInFlight: retainDialClaims(state.ptyDialInFlight, live),
  };
}

export function dropPtyState<Stream, Snapshot>(
  state: TerminalStreamState<Stream, Snapshot>,
  ptyId: string,
): TerminalStreamState<Stream, Snapshot> {
  const dropped = normalizedPtyId(ptyId);
  return retainPtyState(
    state,
    terminalStreamPtyIds(state).filter((id) => id !== dropped),
  );
}

export function retainPtyStateFromReadModel<Stream, Snapshot>(
  state: TerminalStreamState<Stream, Snapshot>,
  readModel: TerminalStreamReadModel | null | undefined,
): TerminalStreamState<Stream, Snapshot> {
  const livePtyIds = livePtyIdsFromReadModel(readModel);
  if (livePtyIds === null) return state;
  return retainPtyState(state, livePtyIds);
}

export function terminalStreamRemovedPtyIds<Stream, Snapshot>(
  state: TerminalStreamState<Stream, Snapshot>,
  livePtyIds: Iterable<string>,
) {
  const live = normalizedPtyIdSet(livePtyIds);
  return terminalStreamPtyIds(state).filter((ptyId) => !live.has(ptyId));
}

export function terminalStreamPtyIds<Stream, Snapshot>(state: TerminalStreamState<Stream, Snapshot>) {
  const ids = new Set<string>();
  addRecordKeys(ids, state.outputChunks);
  addRecordKeys(ids, state.outputChunkStartOffsets);
  addRecordKeys(ids, state.terminalSnapshots);
  addRecordKeys(ids, state.offsets);
  addRecordKeys(ids, state.bottomJumpRevisions);
  addRecordKeys(ids, state.ptyStreams);
  addRecordKeys(ids, state.ptyReconnectTimers);
  addRecordKeys(ids, state.ptyReconnectAttempts);
  for (const ptyId of state.outputFetchInFlight.keys()) addPtyId(ids, ptyId);
  for (const ptyId of state.outputFetchAgain) addPtyId(ids, ptyId);
  for (const claim of state.ptyDialInFlight) addPtyId(ids, ptyIdFromDialClaim(claim));
  return [...ids].sort();
}

export function terminalStreamSizes<Stream, Snapshot>(state: TerminalStreamState<Stream, Snapshot>) {
  return {
    outputChunks: Object.keys(state.outputChunks).length,
    outputChunkStartOffsets: Object.keys(state.outputChunkStartOffsets).length,
    terminalSnapshots: Object.keys(state.terminalSnapshots).length,
    offsets: Object.keys(state.offsets).length,
    bottomJumpRevisions: Object.keys(state.bottomJumpRevisions).length,
    ptyStreams: Object.keys(state.ptyStreams).length,
    ptyReconnectTimers: Object.keys(state.ptyReconnectTimers).length,
    ptyReconnectAttempts: Object.keys(state.ptyReconnectAttempts).length,
    outputFetchInFlight: state.outputFetchInFlight.size,
    outputFetchAgain: state.outputFetchAgain.size,
    ptyDialInFlight: state.ptyDialInFlight.size,
  };
}

function livePtyIdsFromReadModel(readModel: TerminalStreamReadModel | null | undefined) {
  if (!readModel || readModel.ptys == null) return null;
  if (readModel.ptys.length === 0) return [];
  const ids = readModel.ptys.map((pty) => normalizedPtyId(pty?.id ?? "")).filter(Boolean);
  return ids.length > 0 ? ids : null;
}

function retainRecord<T>(record: Record<string, T>, live: ReadonlySet<string>) {
  const retained: Record<string, T> = {};
  for (const [ptyId, value] of Object.entries(record)) {
    if (live.has(ptyId)) retained[ptyId] = value;
  }
  return retained;
}

function retainMap<T>(map: ReadonlyMap<string, T>, live: ReadonlySet<string>) {
  const retained = new Map<string, T>();
  for (const [ptyId, value] of map) {
    if (live.has(ptyId)) retained.set(ptyId, value);
  }
  return retained;
}

function retainSet(set: ReadonlySet<string>, live: ReadonlySet<string>) {
  const retained = new Set<string>();
  for (const ptyId of set) {
    if (live.has(ptyId)) retained.add(ptyId);
  }
  return retained;
}

function retainDialClaims(claims: ReadonlySet<string>, live: ReadonlySet<string>) {
  const retained = new Set<string>();
  for (const claim of claims) {
    if (live.has(ptyIdFromDialClaim(claim))) retained.add(claim);
  }
  return retained;
}

function addRecordKeys(ids: Set<string>, record: Record<string, unknown>) {
  for (const ptyId of Object.keys(record)) addPtyId(ids, ptyId);
}

function addPtyId(ids: Set<string>, ptyId: string) {
  const normalized = normalizedPtyId(ptyId);
  if (normalized) ids.add(normalized);
}

function normalizedPtyIdSet(ptyIds: Iterable<string>) {
  const ids = new Set<string>();
  for (const ptyId of ptyIds) addPtyId(ids, ptyId);
  return ids;
}

function normalizedPtyId(ptyId: string) {
  return ptyId.trim();
}

function ptyIdFromDialClaim(claim: string) {
  const separator = claim.indexOf(":");
  return separator === -1 ? claim : claim.slice(separator + 1);
}
