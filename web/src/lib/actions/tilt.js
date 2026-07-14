/**
 * Subtle cursor-follow tilt for hoverable cards. Rotation is capped small
 * (MAX_DEG) to stay "lieve" rather than a gimmicky 3D effect.
 */
const MAX_DEG = 6;

export function tilt(node) {
  function onMove(e) {
    const rect = node.getBoundingClientRect();
    const px = (e.clientX - rect.left) / rect.width;  // 0..1
    const py = (e.clientY - rect.top) / rect.height;
    const rotateY = (px - 0.5) * 2 * MAX_DEG;
    const rotateX = (0.5 - py) * 2 * MAX_DEG;
    node.style.transition = 'none';
    node.style.transform = `perspective(600px) rotateX(${rotateX}deg) rotateY(${rotateY}deg) scale(1.03)`;
  }

  function onLeave() {
    node.style.transition = 'transform 250ms ease';
    node.style.transform = '';
  }

  node.addEventListener('mousemove', onMove);
  node.addEventListener('mouseleave', onLeave);

  return {
    destroy() {
      node.removeEventListener('mousemove', onMove);
      node.removeEventListener('mouseleave', onLeave);
    }
  };
}
