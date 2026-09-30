import test from 'node:test'
import assert from 'node:assert/strict'
import { resolveAtlasPlanTags } from '../src/utils/atlasPlanTags.ts'

test('merchant recommendation is found beyond the visible tag limit without duplicating the label', () => {
  const tags = ['月度订阅', '独享账号', '人工交付', '推荐', '售后支持']
  assert.deepEqual(resolveAtlasPlanTags(tags), {
    isRecommended: true,
    tags: ['月度订阅', '独享账号', '人工交付'],
  })
  assert.deepEqual(tags, ['月度订阅', '独享账号', '人工交付', '推荐', '售后支持'])
})

test('recommendation supports simplified Chinese, traditional Chinese and English merchant tags', () => {
  for (const tag of ['推荐', '推薦', 'recommended', ' Recommended ']) {
    assert.deepEqual(resolveAtlasPlanTags([tag, '月度订阅']), {
      isRecommended: true,
      tags: ['月度订阅'],
    })
  }
  assert.deepEqual(resolveAtlasPlanTags(['推荐', '推薦', 'recommended']), {
    isRecommended: true,
    tags: [],
  })
})

test('ordinary tags retain their order and limit and do not imply recommendation', () => {
  assert.deepEqual(resolveAtlasPlanTags(['热门', '店长推荐', '月度订阅', '人工交付']), {
    isRecommended: false,
    tags: ['热门', '店长推荐', '月度订阅'],
  })
})

test('removing the merchant tag removes recommendation', () => {
  assert.equal(resolveAtlasPlanTags(['推荐', '月度订阅']).isRecommended, true)
  assert.deepEqual(resolveAtlasPlanTags(['月度订阅']), { isRecommended: false, tags: ['月度订阅'] })
})

test('missing and malformed tags degrade to ordinary cards', () => {
  for (const tags of [undefined, null, '推荐', {}, [], [null, 1, {}, '', ' ']]) {
    assert.deepEqual(resolveAtlasPlanTags(tags), { isRecommended: false, tags: [] })
  }
  assert.deepEqual(resolveAtlasPlanTags([null, '推荐', 1, '独享账号']), {
    isRecommended: true,
    tags: ['独享账号'],
  })
})
