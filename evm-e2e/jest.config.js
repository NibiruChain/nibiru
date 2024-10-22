module.exports = {
    testEnvironment: 'node',
    transform: {
        '^.+\\.(ts|tsx)?$': 'ts-jest',
    },
    testMatch: ['**/test/**/*.test.ts'],
    verbose: true,
    "maxWorkers": 1
};
