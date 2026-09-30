const isRecommendationTag = (tag: string): boolean =>
  ['推荐', '推薦', 'recommended'].includes(tag.trim().toLowerCase())

// 首页将商家配置的推荐标签单独展示，其余标签保留原顺序。
export const resolveAtlasPlanTags = (rawTags: unknown): { isRecommended: boolean; tags: string[] } => {
  const tags = Array.isArray(rawTags)
    ? rawTags.filter((tag): tag is string => typeof tag === 'string' && tag.trim().length > 0)
    : []

  return {
    isRecommended: tags.some(isRecommendationTag),
    tags: tags.filter((tag) => !isRecommendationTag(tag)).slice(0, 3),
  }
}
