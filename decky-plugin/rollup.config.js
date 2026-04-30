import deckyPlugin from '@decky/rollup';
import css from 'rollup-plugin-import-css';

export default deckyPlugin({
  plugins: [css()]
});
