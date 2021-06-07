import _CodeMirror from 'codemirror'
import codemirror from './codemirror.vue'

const CodeMirror = window.CodeMirror || _CodeMirror
const install = (Vue, config) => {
  if (config) {
    if (config.options) {
      codemirror.props.globalOptions.default = () => config.options
    }
    if (config.events) {
      codemirror.props.globalEvents.default = () => config.events
    }
  }
  Vue.component(codemirror.name, codemirror)
}

const VueCodemirror = { CodeMirror, codemirror, install }

export default VueCodemirror
export { CodeMirror, codemirror, install }