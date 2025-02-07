import { useState, useEffect, useRef } from 'react';
import carImage from './assets/car.png';
import './App.css';

function App() {
  const [position, setPosition] = useState(0);
  const [isBusy, setIsBusy] = useState(false);  // <-- new state for button disabling/loading
  const trackRef = useRef(null);
  const carRef = useRef(null);

  const trackWidth = 50000;
  const finishLinePosition = trackWidth - 70;

  useEffect(() => {
    // 1. Fetch initial position from server
    fetch('http://localhost:8080/position')
      .then(res => res.json())
      .then(data => setPosition(data.position))
      .catch(err => console.error("Error fetching initial position:", err));

    // 2. Open WebSocket connection
    const ws = new WebSocket('ws://localhost:8080/ws');

    // On message, parse the updated position
    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.position !== undefined) {
          setPosition(data.position);
        }
      } catch (error) {
        console.error("Error parsing WebSocket message:", error);
      }
    };

    ws.onopen = () => {
      console.log("WebSocket connected");
    };

    ws.onclose = () => {
      console.log("WebSocket disconnected");
    };

    // Clean up WebSocket on unmount
    return () => {
      ws.close();
    };
  }, []);

  // Move the car by sending a delta
  const moveCar = (delta) => {
    // 1. Disable buttons and show loader
    setIsBusy(true);

    fetch('http://localhost:8080/position', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ delta })
    })
    .catch(err => console.error("Error updating position:", err))
    .finally(() => {
      // 2. After request finishes, wait 3 seconds before re-enabling
      setTimeout(() => {
        setIsBusy(false);
      }, 3000);
    });
  };

  const moveForward = () => moveCar(50);
  const moveBackward = () => moveCar(-50);

  // Update car style and scroll position when `position` changes
  useEffect(() => {
    if (carRef.current && trackRef.current) {
      carRef.current.style.left = `${position}px`;

      const visibleTrackStart = trackRef.current.scrollLeft;
      const visibleTrackEnd = visibleTrackStart + trackRef.current.offsetWidth;

      if (position > visibleTrackEnd - carRef.current.offsetWidth * 2 && position < finishLinePosition) {
          trackRef.current.scrollLeft = position - trackRef.current.offsetWidth + carRef.current.offsetWidth * 2;
      } 
      else if (position < visibleTrackStart + carRef.current.offsetWidth) {
          trackRef.current.scrollLeft = Math.max(0, position - carRef.current.offsetWidth);
      }

      if (position >= finishLinePosition) {
        alert("You finished the race!");
      }
    }
  }, [position]);

  // Generate flag markers
  const renderFlags = () => {
    const flags = [];
    for (let i = 5; i <= trackWidth; i += 50) {
      flags.push(
        <div
          key={i}
          className="flag"
          style={{ left: `${i}px`, bottom: '0', transform: 'translateX(-50%)' }}
        >
          { (i/5)*10 }
        </div>
      );
    }
    return flags;
  };

  return (
    <div className="app-container">
      <h1>Car Race (Real-Time)</h1>
      <div className="track-container" ref={trackRef} style={{ width: '80vw' }}>
        <div className="track" style={{ width: `${trackWidth}px` }}>
          <img
            src={carImage}
            alt="Car"
            className="car-image"
            ref={carRef}
            style={{
              position: 'absolute',
              top: '50%',
              transform: 'translateY(-50%)',
              width: '70px',
              height: 'auto'
            }}
          />
          <div className="finish-line" style={{ left: `${finishLinePosition}px` }}></div>
          {renderFlags()}
        </div>
      </div>

      <div className="controls">
        <button onClick={moveForward} disabled={isBusy}>
          {isBusy ? "Loading..." : "Forward"}
        </button>
        <button onClick={moveBackward} disabled={isBusy}>
          {isBusy ? "Loading..." : "Backward"}
        </button>
      </div>
    </div>
  );
}

export default App;
