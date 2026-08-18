vx = Math.random() * 2 - 1;
vy = Math.random() * 5;
vy = 0;
rotationspeed = Math.random() * 10 + 5;
if(random(2) == 0)
{
   rotationspeed *= -1;
}
this.onEnterFrame = function()
{
   _X = _X + vx;
   _Y = _Y + vy;
   _rotation = _rotation + rotationspeed;
   vy += _root.gravity * 1.3;
   if(_Y >= 900 || _root.deleteeverything)
   {
      removeMovieClip(this);
      delete this.onEnterFrame;
   }
};
