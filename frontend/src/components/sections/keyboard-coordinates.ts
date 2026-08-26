import { KeyboardCode, type KeyboardCoordinateGetter } from '@dnd-kit/core';

/**
 * Arrow-key movement for the section board that jumps a whole column per press.
 *
 * dnd-kit's `defaultKeyboardCoordinateGetter` translates the dragged card by
 * 25px per arrow key. A section column is 288px wide (`w-72`), so reaching the
 * neighbouring one would take twelve presses and crossing a six-section board
 * over seventy — which is not a keyboard path anybody would use.
 *
 * This getter instead snaps the card to the centre of the adjacent column, so
 * one press moves one section and the number of presses matches the number of
 * columns the user can see. Up/down are left alone: the columns are ordered
 * horizontally and a card's position within a column carries no meaning, so
 * there is nothing for a vertical key to do.
 */
export const sectionKeyboardCoordinates: KeyboardCoordinateGetter = (
  event,
  { currentCoordinates, context: { collisionRect, droppableContainers, droppableRects } }
) => {
  const step = event.code === KeyboardCode.Right ? 1 : event.code === KeyboardCode.Left ? -1 : 0;
  if (step === 0 || !collisionRect) {
    return undefined;
  }
  event.preventDefault();

  // droppableRects holds the measured rect; the container's own `rect` ref can
  // still be null for a column that has not been measured this drag.
  const columns = droppableContainers
    .getEnabled()
    .map((container) => droppableRects.get(container.id))
    .filter((rect) => rect !== undefined)
    .sort((a, b) => a.left - b.left);
  if (columns.length === 0) {
    return undefined;
  }

  const cardCentre = collisionRect.left + collisionRect.width / 2;
  const currentIndex = nearestColumnIndex(columns, cardCentre);
  const target = columns[clamp(currentIndex + step, 0, columns.length - 1)];

  return {
    x: currentCoordinates.x + (target.left + target.width / 2 - cardCentre),
    y: currentCoordinates.y,
  };
};

/**
 * The column the card is currently over, or — while it sits in the gap between
 * two of them — whichever centre it is closest to.
 */
function nearestColumnIndex(
  columns: { left: number; width: number }[],
  cardCentre: number
): number {
  let best = 0;
  let bestDistance = Infinity;
  for (const [index, column] of columns.entries()) {
    const distance = Math.abs(cardCentre - (column.left + column.width / 2));
    if (distance < bestDistance) {
      bestDistance = distance;
      best = index;
    }
  }
  return best;
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}
