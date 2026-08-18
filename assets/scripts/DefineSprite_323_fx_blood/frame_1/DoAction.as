_rotation = random(135) - 90;
if(random(2) == 0)
{
   _rotation = _rotation + 180;
}
speed = Math.random() * 5 + 5;
dirx = Math.cos(_rotation * 3.141592653589793 / 180) * speed;
diry = Math.sin(_rotation * 3.141592653589793 / 180) * speed;
_X = _X + dirx * 1;
_Y = _Y + diry * 1;
_xscale = _xscale - random(60);
_yscale = _xscale;
time = 0;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      speed += 0.5;
      if(_rotation > -90 && _rotation < 90)
      {
         _rotation = _rotation + 4;
      }
      else
      {
         _rotation = _rotation - 4;
      }
      dirx = Math.cos(_rotation * 3.141592653589793 / 180) * speed;
      diry = Math.sin(_rotation * 3.141592653589793 / 180) * speed;
      _X = _X + dirx;
      _Y = _Y + diry;
      if(_Y >= 900 || _alpha <= 1 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
