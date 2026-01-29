// Client-side Logger Interceptor
// Redirects (or duplicates) console logs to the server.

const originalLog = console.log;
const originalWarn = console.warn;
const originalError = console.error;
const originalInfo = console.info;

const sendLog = (level, args) => {
    try {
        // Convert args to a single string or relevant object
        const message = args.map(arg => {
            if (typeof arg === 'object') {
                try {
                    return JSON.stringify(arg);
                } catch (e) {
                    return String(arg);
                }
            }
            return String(arg);
        }).join(' ');

        fetch('/sqliter/logs', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                level,
                message,
            }),
        }).catch(() => {
            // Prevent infinite loop if logging fails
        });
    } catch (e) {
        // Ignore internal logging errors
    }
};

export const initLogger = () => {
    console.log = (...args) => {
        originalLog.apply(console, args);
        sendLog('info', args);
    };

    console.warn = (...args) => {
        originalWarn.apply(console, args);
        sendLog('warn', args);
    };

    console.error = (...args) => {
        originalError.apply(console, args);
        sendLog('error', args);
    };

    console.info = (...args) => {
        originalInfo.apply(console, args);
        sendLog('info', args);
    };
    
    // Initial log to verify connection
    console.info("Client Logger Initialized");
};
