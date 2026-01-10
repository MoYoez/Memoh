export const time = (params: {
  date: Date
  locale?: Intl.LocalesArgument
}) => {
  return `
date: ${params.date.toLocaleDateString(params.locale)}
time: ${params.date.toLocaleTimeString(params.locale)}
  `.trim()
}