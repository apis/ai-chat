import cssnano from "cssnano"
import postcssImport from "postcss-import"
import postcssInherit from "postcss-inherit"


export default {
    base: "/chat",
    css: {
        postcss: {
            plugins: [
                cssnano({preset: "default"}),
                postcssImport(),
                postcssInherit(),
            ],
        }
    },
}