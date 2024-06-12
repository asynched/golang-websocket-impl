const ws = new WebSocket('ws://localhost:8080/ws/echo')

ws.addEventListener('open', async () => {
  console.log('Connected to the server')

  const contents = await Bun.file('main.go').text()

  setInterval(() => ws.send(contents), 1_000)
})

ws.addEventListener('message', ({ data }) => console.log(data))

ws.addEventListener('close', ({ reason }) => {
  console.warn(`Connection closed: ${reason}`)
  process.exit(0)
})

ws.addEventListener('error', (error) => {
  console.error(error)
  process.exit(1)
})
