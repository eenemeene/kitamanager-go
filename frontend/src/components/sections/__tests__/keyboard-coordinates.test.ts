import { sectionKeyboardCoordinates } from '../keyboard-coordinates';

/**
 * The board's columns are 288px wide, so the interesting property is that one
 * key press moves exactly one column — the default getter's 25px nudge is what
 * made the keyboard path unusable in the first place.
 */

const COLUMN_WIDTH = 288;
const GAP = 16;

/** Three columns laid out left to right, as the board renders them. */
function columnRect(index: number) {
  return {
    left: index * (COLUMN_WIDTH + GAP),
    width: COLUMN_WIDTH,
    top: 0,
    height: 600,
    right: index * (COLUMN_WIDTH + GAP) + COLUMN_WIDTH,
    bottom: 600,
  };
}

/**
 * A context with three columns and the dragged card centred over `overColumn`.
 * Only the fields the getter reads are populated.
 */
function contextWith(overColumn: number) {
  const rects = [columnRect(0), columnRect(1), columnRect(2)];
  const cardWidth = 256;
  const centre = rects[overColumn].left + COLUMN_WIDTH / 2;

  return {
    active: 'child-1',
    currentCoordinates: { x: 0, y: 0 },
    context: {
      collisionRect: {
        left: centre - cardWidth / 2,
        width: cardWidth,
        top: 0,
        height: 64,
        right: centre + cardWidth / 2,
        bottom: 64,
      },
      droppableRects: new Map(rects.map((rect, i) => [String(i + 1), rect])),
      droppableContainers: {
        getEnabled: () => rects.map((_, i) => ({ id: String(i + 1) })),
      },
    },
  } as unknown as Parameters<typeof sectionKeyboardCoordinates>[1];
}

function press(code: string): KeyboardEvent {
  return { code, preventDefault: jest.fn() } as unknown as KeyboardEvent;
}

/** Where the card's centre ends up after applying the returned translation. */
function resultingCentre(overColumn: number, code: string): number | undefined {
  const args = contextWith(overColumn);
  const result = sectionKeyboardCoordinates(press(code), args);
  if (!result) return undefined;
  const { collisionRect } = args.context;
  return collisionRect!.left + collisionRect!.width / 2 + (result.x - args.currentCoordinates.x);
}

describe('sectionKeyboardCoordinates', () => {
  it('moves one whole column to the right per press', () => {
    expect(resultingCentre(0, 'ArrowRight')).toBe(columnRect(1).left + COLUMN_WIDTH / 2);
  });

  it('moves one whole column to the left per press', () => {
    expect(resultingCentre(2, 'ArrowLeft')).toBe(columnRect(1).left + COLUMN_WIDTH / 2);
  });

  it('stops at the last column instead of wrapping around', () => {
    expect(resultingCentre(2, 'ArrowRight')).toBe(columnRect(2).left + COLUMN_WIDTH / 2);
  });

  it('stops at the first column instead of wrapping around', () => {
    expect(resultingCentre(0, 'ArrowLeft')).toBe(columnRect(0).left + COLUMN_WIDTH / 2);
  });

  it.each(['ArrowUp', 'ArrowDown', 'KeyA'])(
    'leaves %s to dnd-kit, since the columns are ordered horizontally',
    (code) => {
      expect(sectionKeyboardCoordinates(press(code), contextWith(1))).toBeUndefined();
    }
  );

  it('claims the arrow key so the board does not scroll under the drag', () => {
    const event = press('ArrowRight');
    sectionKeyboardCoordinates(event, contextWith(0));
    expect(event.preventDefault).toHaveBeenCalled();
  });
});
