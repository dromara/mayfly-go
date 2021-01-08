<template>
  <div style="height: 600px" id="xterm" class="xterm" />
</template>

<script>
import 'xterm/css/xterm.css'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'

export default {
  name: 'Xterm',
  props: {
    socketURI: {
      type: String,
      default: '',
    },
  },
  watch: {
    socketURI(val) {
      if (val !== '') {
        this.initSocket()
      }
    },
  },
  mounted() {
    this.initSocket()
    // this.initTerm()
  },
  beforeDestroy() {
    this.socket.close()
    this.term.dispose()
  },
  methods: {
    initXterm() {
      const term = new Terminal({
        fontSize: 14,
        cursorBlink: true,
        // cursorStyle: 'underline', //光标样式
        disableStdin: false,
        theme: {
        foreground: "#7e9192", //字体
        background: "#002833", //背景色
        cursor: "help", //设置光标
        lineHeight: 16
      }
      })
      const fitAddon = new FitAddon()
      term.loadAddon(fitAddon)
      term.open(document.getElementById('xterm'))
      fitAddon.fit()
      term.focus()
      this.term = term

      // / **
      //     *添加事件监听器，用于按下键时的事件。事件值包含
      //     *将在data事件以及DOM事件中发送的字符串
      //     *触发了它。
      //     * @返回一个IDisposable停止监听。
      //  * /
      //   / ** 更新：xterm 4.x（新增）
      //  *为数据事件触发时添加事件侦听器。发生这种情况
      //  *用户输入或粘贴到终端时的示例。事件值
      //  *是`string`结果的结果，在典型的设置中，应该通过
      //  *到支持pty。
      //  * @返回一个IDisposable停止监听。
      //  * /
      // 支持输入与粘贴方法
      term.onData((key) => {
        const cmd = {
          type: 'cmd',
          msg: key,
        }
        this.send(cmd)
      })
      // 为解决窗体resize方法才会向后端发送列数和行数，所以页面加载时也要触发此方法
      this.send({
        type: 'resize',
        Cols: parseInt(term.cols),
        Rows: parseInt(term.rows),
      })
    },
    initSocket() {
      this.socket = new WebSocket(this.socketURI)
      // 监听socket连接
      this.socket.onopen = this.open
      // 监听socket错误信息
      this.socket.onerror = this.error
      // 监听socket消息
      this.socket.onmessage = this.getMessage
      // 发送socket消息
      this.socket.onsend = this.send
    },
    open: function () {
      console.log('socket连接成功')
      this.initXterm()
      //开启心跳
      //   this.start();
    },
    error: function () {
      console.log('连接错误')
      //重连
      this.reconnect()
    },
    close: function () {
      this.socket.close()
      console.log('socket已经关闭')
      //重连
      //   this.reconnect()
    },
    getMessage: function (msg) {
    //   console.log(msg)
      this.term.write(msg['data'])
      //msg是返回的数据
      //   msg = JSON.parse(msg.data);
      //   this.socket.send("ping");//有事没事ping一下，看看ws还活着没
      //   //switch用于处理返回的数据，根据返回数据的格式去判断
      //   switch (msg["operation"]) {
      //     case "stdout":
      //       this.term.write(msg["data"]);//这里write也许不是固定的，失败后找后端看一下该怎么往term里面write
      //       break;
      //     default:
      //       console.error("Unexpected message type:", msg);//但是错误是固定的。。。。
      //   }
      //收到服务器信息，心跳重置
      //   this.reset();
    },
    send: function (msg) {
        // console.log(msg)
      this.socket.send(JSON.stringify(msg))
    },
    closeAll() {
      this.close()
      this.term.dispose()
      this.term = null
    },
  },
}
</script>