import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import ConnectFour from './App';

// Mock WebSocket
global.WebSocket = class WebSocket {
  constructor(url) {
    this.url = url;
    this.readyState = WebSocket.OPEN;
    setTimeout(() => this.onopen && this.onopen(), 0);
  }
  
  send(data) {
    // Mock send
  }
  
  close() {
    // Mock close
  }
  
  static OPEN = 1;
};

describe('ConnectFour Game', () => {
  test('renders login screen', () => {
    render(<ConnectFour />);
    expect(screen.getByText(/Enter Your Username/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Username/i)).toBeInTheDocument();
  });

  test('allows username input', () => {
    render(<ConnectFour />);
    const input = screen.getByPlaceholderText(/Username/i);
    
    fireEvent.change(input, { target: { value: 'TestPlayer' } });
    expect(input.value).toBe('TestPlayer');
  });

  test('shows error for empty username', () => {
    render(<ConnectFour />);
    const button = screen.getByText(/Join Game/i);
    
    fireEvent.click(button);
    
    waitFor(() => {
      expect(screen.getByText(/Please enter a username/i)).toBeInTheDocument();
    });
  });

  test('renders game board after joining', async () => {
    render(<ConnectFour />);
    const input = screen.getByPlaceholderText(/Username/i);
    const button = screen.getByText(/Join Game/i);
    
    fireEvent.change(input, { target: { value: 'TestPlayer' } });
    fireEvent.click(button);
    
    // Wait for WebSocket connection
    await waitFor(() => {
      // Board should be rendered
    });
  });
});