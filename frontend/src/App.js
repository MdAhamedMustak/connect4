import React, { useState, useEffect, useRef } from 'react';
import { Trophy, Users, Clock, Wifi, WifiOff } from 'lucide-react';

const ROWS = 6;
const COLS = 7;

export default function ConnectFour() {
  const [username, setUsername] = useState('');
  const [gameState, setGameState] = useState('login'); // login, waiting, playing, finished
  const [board, setBoard] = useState(Array(ROWS).fill(null).map(() => Array(COLS).fill(null)));
  const [currentPlayer, setCurrentPlayer] = useState(null);
  const [myColor, setMyColor] = useState(null);
  const [opponent, setOpponent] = useState('');
  const [winner, setWinner] = useState(null);
  const [message, setMessage] = useState('');
  const [leaderboard, setLeaderboard] = useState([]);
  const [connected, setConnected] = useState(false);
  const [gameId, setGameId] = useState(null);
  const [timeLeft, setTimeLeft] = useState(10);
  
  const ws = useRef(null);

  useEffect(() => {
    if (gameState === 'waiting' && timeLeft > 0) {
      const timer = setTimeout(() => setTimeLeft(timeLeft - 1), 1000);
      return () => clearTimeout(timer);
    }
  }, [gameState, timeLeft]);

  // Cleanup WebSocket on unmount
  useEffect(() => {
    return () => {
      if (ws.current) {
        console.log('Cleaning up WebSocket connection');
        ws.current.close();
      }
    };
  }, []);

  const connectWebSocket = () => {
    // Close existing connection if any
    if (ws.current) {
      ws.current.close();
    }

    // Determine WebSocket URL
    let wsUrl;
    if (process.env.REACT_APP_WS_URL) {
      wsUrl = process.env.REACT_APP_WS_URL;
    } else {
      // Fallback to localhost
      wsUrl = 'ws://localhost:8080/ws';
    }
    
    console.log('Connecting to WebSocket:', wsUrl);
    ws.current = new WebSocket(wsUrl);
    
    ws.current.onopen = () => {
      setConnected(true);
      console.log('✓ WebSocket connected');
    };
    
    ws.current.onmessage = (event) => {
      console.log('📨 Received:', event.data);
      try {
        const data = JSON.parse(event.data);
        handleMessage(data);
      } catch (e) {
        console.error('Failed to parse message:', e);
      }
    };
    
    ws.current.onerror = (error) => {
      console.error('❌ WebSocket error:', error);
      setConnected(false);
      setMessage('Connection error. Please refresh the page.');
    };
    
    ws.current.onclose = () => {
      setConnected(false);
      console.log('❌ WebSocket disconnected');
      
      // Don't auto-reconnect during game to avoid issues
      if (gameState === 'login') {
        setTimeout(() => {
          console.log('Attempting to reconnect...');
          connectWebSocket();
        }, 3000);
      }
    };
  };

  const handleMessage = (data) => {
    console.log('=== Received message ===', data); // Debug log
    
    switch (data.type) {
      case 'waiting':
        setGameState('waiting');
        setMessage('Waiting for opponent...');
        setTimeLeft(10);
        break;
        
      case 'game_start':
        console.log('GAME START:', data);
        console.log('Setting myColor to:', data.color);
        
        setGameState('playing');
        setMyColor(data.color);
        setOpponent(data.opponent);
        setCurrentPlayer(data.current_player);
        setGameId(data.game_id);
        setMessage(`Game started! You are ${data.color.toUpperCase()}. ${data.current_player === data.color ? 'Your turn!' : 'Opponent\'s turn'}`);
        setBoard(Array(ROWS).fill(null).map(() => Array(COLS).fill(null)));
        setWinner(null);
        break;
        
      case 'move':
        console.log('Move update received:', data);
        console.log('Board before update:', board);
        console.log('New board:', data.board);
        
        if (data.board && Array.isArray(data.board)) {
          setBoard(data.board);
          setCurrentPlayer(data.current_player);
          
          if (data.current_player === myColor) {
            setMessage('Your turn!');
          } else {
            setMessage('Opponent is thinking...');
          }
        } else {
          console.error('Invalid board data received');
        }
        break;
        
      case 'game_over':
        console.log('======================');
        console.log('GAME OVER RECEIVED');
        console.log('======================');
        console.log('Full data object:', JSON.stringify(data, null, 2));
        console.log('Winner from data:', data.winner);
        console.log('Winner type:', typeof data.winner);
        console.log('My color:', myColor);
        console.log('My color type:', typeof myColor);
        console.log('Are they equal?', data.winner === myColor);
        console.log('String comparison:', String(data.winner) === String(myColor));
        console.log('======================');
        
        setBoard(data.board);
        setWinner(data.winner);
        setGameState('finished');
        
        // The winner comes as a color string ("red" or "yellow")
        const winnerColor = String(data.winner).toLowerCase();
        const playerColor = String(myColor).toLowerCase();
        
        console.log('After conversion:');
        console.log('Winner color:', winnerColor);
        console.log('Player color:', playerColor);
        console.log('Match?', winnerColor === playerColor);
        
        if (winnerColor === 'draw') {
          setMessage("It's a draw! 🤝");
        } else if (winnerColor === playerColor) {
          setMessage('🎉 You Won! Congratulations! 🏆');
        } else {
          setMessage('You Lost! Better luck next time! 😔');
        }
        
        // Refresh leaderboard
        setTimeout(() => {
          fetchLeaderboard();
        }, 1000);
        break;
        
      case 'opponent_disconnected':
        setMessage('Opponent disconnected. Waiting 30s for reconnection...');
        break;
        
      case 'game_forfeited':
        setGameState('finished');
        setMessage('Opponent forfeited. You win!');
        fetchLeaderboard();
        break;
        
      case 'error':
        setMessage('Error: ' + data.message);
        console.error('Game error:', data.message);
        break;
        
      default:
        console.log('Unknown message type:', data.type);
    }
  };

  const joinGame = () => {
    if (!username.trim()) {
      setMessage('Please enter a username');
      return;
    }
    
    console.log('Joining game as:', username);
    connectWebSocket();
    
    // Wait for connection to open before sending join message
    const checkConnection = setInterval(() => {
      if (ws.current && ws.current.readyState === WebSocket.OPEN) {
        clearInterval(checkConnection);
        console.log('Sending join message');
        ws.current.send(JSON.stringify({
          type: 'join',
          username: username
        }));
      }
    }, 100);
    
    // Timeout after 5 seconds
    setTimeout(() => {
      clearInterval(checkConnection);
      if (!connected) {
        setMessage('Failed to connect. Is the backend running?');
        console.error('Connection timeout');
      }
    }, 5000);
  };

  const makeMove = (col) => {
    console.log('Attempting move:', { col, gameState, currentPlayer, myColor }); // Debug log
    
    if (gameState !== 'playing') {
      console.log('Game not in playing state');
      setMessage('Game is not active');
      return;
    }
    
    if (currentPlayer !== myColor) {
      console.log('Not your turn');
      setMessage('Wait for your turn!');
      return;
    }
    
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.log('WebSocket not connected');
      setMessage('Connection lost. Reconnecting...');
      return;
    }
    
    console.log('Sending move to server:', col);
    ws.current.send(JSON.stringify({
      type: 'move',
      column: col
    }));
  };

  const playAgain = () => {
    setGameState('login');
    setBoard(Array(ROWS).fill(null).map(() => Array(COLS).fill(null)));
    setWinner(null);
    setMessage('');
    if (ws.current) {
      ws.current.close();
    }
  };

  const fetchLeaderboard = async () => {
    try {
      const response = await fetch(`${window.location.protocol}//${window.location.hostname}:8080/leaderboard`);
      const data = await response.json();
      setLeaderboard(data || []);
    } catch (error) {
      console.error('Failed to fetch leaderboard:', error);
    }
  };

  useEffect(() => {
    fetchLeaderboard();
  }, []);

  const getCellColor = (cell) => {
    if (cell === 'red') return 'bg-red-500';
    if (cell === 'yellow') return 'bg-yellow-400';
    return 'bg-white';
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-900 via-blue-800 to-purple-900 p-8">
      <div className="max-w-6xl mx-auto">
        <div className="text-center mb-8">
          <h1 className="text-5xl font-bold text-white mb-2">🎯 4 in a Row</h1>
          <div className="flex items-center justify-center gap-2 text-white">
            {connected ? <Wifi className="w-5 h-5 text-green-400" /> : <WifiOff className="w-5 h-5 text-red-400" />}
            <span className="text-sm">{connected ? 'Connected' : 'Disconnected'}</span>
          </div>
        </div>

        {gameState === 'login' && (
          <div className="bg-white rounded-lg shadow-2xl p-8 max-w-md mx-auto">
            <h2 className="text-2xl font-bold text-gray-800 mb-4">Enter Your Username</h2>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              onKeyPress={(e) => e.key === 'Enter' && joinGame()}
              placeholder="Username"
              className="w-full px-4 py-3 border border-gray-300 rounded-lg mb-4 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              onClick={joinGame}
              className="w-full bg-blue-600 text-white py-3 rounded-lg font-semibold hover:bg-blue-700 transition"
            >
              Join Game
            </button>
          </div>
        )}

        {gameState === 'waiting' && (
          <div className="bg-white rounded-lg shadow-2xl p-8 max-w-md mx-auto text-center">
            <Users className="w-16 h-16 mx-auto mb-4 text-blue-600 animate-pulse" />
            <h2 className="text-2xl font-bold text-gray-800 mb-2">Waiting for Opponent</h2>
            <div className="flex items-center justify-center gap-2 text-gray-600 mb-4">
              <Clock className="w-5 h-5" />
              <span>Bot joins in {timeLeft}s if no player found</span>
            </div>
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          </div>
        )}

        {(gameState === 'playing' || gameState === 'finished') && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
            <div className="lg:col-span-2">
              <div className="bg-white rounded-lg shadow-2xl p-6">
                {/* Debug Info - Remove this after testing */}
                <div className="mb-4 p-2 bg-gray-100 rounded text-xs">
                  <div>State: {gameState}</div>
                  <div>Your Color: {myColor || 'none'}</div>
                  <div>Current Turn: {currentPlayer || 'none'}</div>
                  <div>Can Move: {(gameState === 'playing' && currentPlayer === myColor) ? 'YES' : 'NO'}</div>
                  <div>WS Status: {connected ? 'Connected' : 'Disconnected'}</div>
                </div>

                <div className="mb-4 flex justify-between items-center">
                  <div className="text-lg font-semibold">
                    <span className={myColor === 'red' ? 'text-red-600' : 'text-yellow-600'}>
                      You ({username})
                    </span>
                    <span className="mx-2">vs</span>
                    <span className={myColor === 'red' ? 'text-yellow-600' : 'text-red-600'}>
                      {opponent}
                    </span>
                  </div>
                  {gameState === 'playing' && (
                    <div className={`px-4 py-2 rounded-lg font-semibold ${
                      currentPlayer === myColor ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                    }`}>
                      {currentPlayer === myColor ? 'Your Turn' : "Opponent's Turn"}
                    </div>
                  )}
                </div>

                {message && (
                  <div className={`mb-4 p-3 rounded-lg text-center font-semibold ${
                    message.includes('won') || message.includes('win') ? 'bg-green-100 text-green-800' :
                    message.includes('lost') ? 'bg-red-100 text-red-800' :
                    'bg-blue-100 text-blue-800'
                  }`}>
                    {message}
                  </div>
                )}

                <div className="bg-blue-600 rounded-lg p-4 inline-block">
                  {/* Column click area above board */}
                  <div className="grid gap-2 mb-2" style={{ gridTemplateColumns: `repeat(${COLS}, 1fr)` }}>
                    {Array(COLS).fill(0).map((_, colIdx) => (
                      <div
                        key={`col-${colIdx}`}
                        onClick={() => makeMove(colIdx)}
                        className={`w-16 h-8 flex items-center justify-center text-white font-bold ${
                          gameState === 'playing' && currentPlayer === myColor 
                            ? 'cursor-pointer hover:bg-blue-500 rounded-t-lg transition' 
                            : 'cursor-not-allowed opacity-50'
                        }`}
                      >
                        {gameState === 'playing' && currentPlayer === myColor && '↓'}
                      </div>
                    ))}
                  </div>
                  
                  {/* Game board */}
                  <div className="grid gap-2" style={{ gridTemplateColumns: `repeat(${COLS}, 1fr)` }}>
                    {board.map((row, rowIdx) =>
                      row.map((cell, colIdx) => (
                        <div
                          key={`${rowIdx}-${colIdx}`}
                          className={`w-16 h-16 rounded-full ${getCellColor(cell)} border-4 border-blue-700 shadow-inner`}
                        />
                      ))
                    )}
                  </div>
                </div>

                {gameState === 'finished' && (
                  <button
                    onClick={playAgain}
                    className="mt-4 w-full bg-blue-600 text-white py-3 rounded-lg font-semibold hover:bg-blue-700 transition"
                  >
                    Play Again
                  </button>
                )}
              </div>
            </div>

            <div className="bg-white rounded-lg shadow-2xl p-6">
              <div className="flex items-center gap-2 mb-4">
                <Trophy className="w-6 h-6 text-yellow-500" />
                <h3 className="text-xl font-bold text-gray-800">Leaderboard</h3>
              </div>
              <div className="space-y-2">
                {leaderboard.length === 0 ? (
                  <p className="text-gray-500 text-center py-4">No games played yet</p>
                ) : (
                  leaderboard.map((player, idx) => (
                    <div key={idx} className="flex justify-between items-center p-3 bg-gray-50 rounded-lg">
                      <div className="flex items-center gap-2">
                        <span className="text-lg font-bold text-gray-600">#{idx + 1}</span>
                        <span className={`font-semibold ${player.username === username ? 'text-blue-600' : 'text-gray-800'}`}>
                          {player.username}
                        </span>
                      </div>
                      <span className="text-lg font-bold text-green-600">{player.wins} wins</span>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}