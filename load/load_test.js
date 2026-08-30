import grpc from 'k6/net/grpc';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const gamesWon   = new Counter('games_won');
const gamesDrawn = new Counter('games_drawn');
const gameErrors = new Counter('game_errors');

const client = new grpc.Client();
// Path is relative to this script file
client.load(['../proto/voodoo/v1'], 'voodoo.proto');

// 500k games × 2 players each = 1 000 000 unique players
export const options = {
  scenarios: {
    million_players: {
      executor:    'shared-iterations',
      vus:         5000,
      iterations:  500_000,
      maxDuration: '10m',
    },
  },
  thresholds: {
    grpc_req_duration: ['p(95)<200'],
    game_errors:       ['count<1000'],
  },
};

const ADDR = __ENV.SERVER_ADDR || 'localhost:8080';

export default function () {
  client.connect(ADDR, { plaintext: true });

  const p1 = `p1-vu${__VU}-i${__ITER}`;
  const p2 = `p2-vu${__VU}-i${__ITER}`;

  // --- CreateGame ---
  const created = client.invoke('voodoo.v1.VoodooService/CreateGame', {
    player_id: p1,
  });
  if (!check(created, { 'CreateGame ok': (r) => r.status === grpc.StatusOK })) {
    gameErrors.add(1);
    client.close();
    return;
  }
  const gameId = created.message.gameId;

  // --- JoinGame ---
  const joined = client.invoke('voodoo.v1.VoodooService/JoinGame', {
    game_id:   gameId,
    player_id: p2,
  });
  if (!check(joined, { 'JoinGame ok': (r) => r.status === grpc.StatusOK })) {
    gameErrors.add(1);
    client.close();
    return;
  }

  // --- Play moves (cells 0-8 in order, alternating players) ---
  // With a 3×3 board this sequence ends at cell 6: p1 wins via diagonal 2-4-6
  const players = [p1, p2];
  let winner = '';
  for (let cell = 0; cell < 9 && winner === ''; cell++) {
    const move = client.invoke('voodoo.v1.VoodooService/UpdateGame', {
      game_id:   gameId,
      player_id: players[cell % 2],
      cell:      cell,
    });
    if (!check(move, { 'UpdateGame ok': (r) => r.status === grpc.StatusOK })) {
      gameErrors.add(1);
      break;
    }
    winner = move.message.winner ?? '';
  }

  if      (winner === 'draw') gamesDrawn.add(1);
  else if (winner !== '')     gamesWon.add(1);

  client.close();
}
